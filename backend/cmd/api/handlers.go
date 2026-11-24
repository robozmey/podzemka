package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/overpass"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

type createCityRequest struct {
	Name     string             `json:"name"`
	Slug     string             `json:"slug"`
	Stations []domain.Station   `json:"stations"`
	Lines    []domain.MetroLine `json:"lines"`
}

var validCitySlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Hello, World!"))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (app application) citiesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cities, err := app.repo.ListCities(ctx)
	if err != nil {
		log.Printf("list cities: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось загрузить города",
		})
		return
	}

	writeJSON(w, http.StatusOK, cities)
}

func catalogCitiesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, recommendedCities)
}

func (app application) previewImportHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Не удалось прочитать файл Overpass (максимум 10 МБ)",
		})
		return
	}
	defer r.MultipartForm.RemoveAll()

	name := strings.TrimSpace(r.FormValue("name"))
	slug := ""
	if name != "" {
		if len([]rune(name)) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Название города должно содержать не более 100 символов",
			})
			return
		}
		slug = citySlug(name)
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Не удалось сформировать slug города",
			})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		nameExists, err := app.repo.CityExistsByName(ctx, name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось проверить город"})
			return
		}
		if nameExists {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Город с таким названием уже добавлен"})
			return
		}
		slugExists, err := app.repo.CityExists(ctx, slug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось проверить slug города"})
			return
		}
		if slugExists {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Город с таким slug уже существует"})
			return
		}
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Загрузите JSON-файл Overpass в поле file",
		})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не удалось прочитать JSON-файл"})
		return
	}
	// PowerShell and some Windows editors may save UTF-8 JSON with a BOM.
	// encoding/json does not accept the BOM as the first byte, although it is
	// harmless for a JSON import, so remove it before parsing.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var schema domain.MetroContent
	if err := json.Unmarshal(data, &schema); err != nil || schema.Stations == nil || schema.Lines == nil {
		schema, err = overpass.Parse(bytes.NewReader(data), overpass.MapWidth, overpass.MapHeight)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Не удалось разобрать JSON-файл схемы или Overpass",
			})
			return
		}
	}

	if err := schema.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Парсер сформировал некорректную схему",
		})
		return
	}
	if len(schema.Stations) == 0 || len(schema.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Файл не содержит непустую схему метро",
		})
		return
	}
	if len(schema.Stations) >= 20 && len(schema.Lines) < 2 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Файл похож на ошибочную схему: для большого города найдена только одна линия. Загрузите исходный Overpass JSON заново",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":          name,
		"slug":          slug,
		"stationsCount": len(schema.Stations),
		"linesCount":    len(schema.Lines),
		"schema":        schema,
	})
}

func (app application) createCityHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request createCityRequest
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Некорректное тело запроса",
		})
		return
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "В запросе должен быть один JSON-объект",
		})
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Slug = strings.TrimSpace(request.Slug)

	if request.Name == "" || !validCitySlug.MatchString(request.Slug) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Укажите название города и slug в формате lowercase-hyphen",
		})
		return
	}

	content := domain.MetroContent{
		Stations: request.Stations,
		Lines:    request.Lines,
	}

	if err := content.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Некорректная схема метро",
		})
		return
	}
	if len(content.Stations) == 0 || len(content.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Схема метро должна содержать станции и линии",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	nameExists, err := app.repo.CityExistsByName(ctx, request.Name)
	if err != nil {
		log.Printf("check city name before create: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось проверить город",
		})
		return
	}
	if nameExists {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Город с таким названием уже добавлен",
		})
		return
	}

	exists, err := app.repo.CityExists(ctx, request.Slug)
	if err != nil {
		log.Printf("check city before create: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось проверить город",
		})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Город с таким slug уже существует",
		})
		return
	}

	schema := domain.MetroSchema{
		City: domain.City{
			Name: request.Name,
			Slug: request.Slug,
		},
		Stations: content.Stations,
		Lines:    content.Lines,
	}

	if err := app.repo.SaveSchema(ctx, schema); err != nil {
		log.Printf("save city schema: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось сохранить город",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"name": request.Name,
		"slug": request.Slug,
	})
}

func (app application) schemaHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	schema, err := app.repo.GetSchemaBySlug(ctx, r.PathValue("citySlug"))

	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Схема города не найдена",
		})
		return
	}

	if err != nil {
		log.Printf("get schema: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось загрузить схему",
		})
		return
	}

	writeJSON(w, http.StatusOK, schema)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
