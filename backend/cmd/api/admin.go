package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/robozmey/podzemka/backend/internal/overpass"
)

func adminOnly(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(token) == "" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Администрирование не настроено на сервере",
			})
			return
		}

		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Требуется токен администратора",
			})
			return
		}

		expectedHash := sha256.Sum256([]byte(token))
		providedHash := sha256.Sum256([]byte(parts[1]))
		if subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Неверный токен администратора",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func adminSessionHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (app application) previewCityHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request struct {
		Name string `json:"name"`
	}
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Укажите название города"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "В запросе должен быть один JSON-объект"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len([]rune(request.Name)) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Название города должно содержать от 1 до 100 символов"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 160*time.Second)
	defer cancel()
	baseSlug := citySlug(request.Name)
	if baseSlug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не удалось сформировать адрес города из названия"})
		return
	}
	nameExists, err := app.repo.CityExistsByName(ctx, request.Name)
	if err != nil {
		log.Printf("check city name %q: %v", request.Name, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось проверить существование города"})
		return
	}
	if nameExists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("Город «%s» уже добавлен", request.Name)})
		return
	}
	slugExists, err := app.repo.CityExists(ctx, baseSlug)
	if err != nil {
		log.Printf("check city slug %q: %v", baseSlug, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось проверить адрес города"})
		return
	}
	if slugExists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("Город с адресом «%s» уже добавлен", baseSlug)})
		return
	}
	previewCity := app.previewCity
	if previewCity == nil {
		previewCity = overpass.FetchCity
	}
	content, err := previewCity(ctx, request.Name)
	if err != nil {
		log.Printf("preview city %q: %v", request.Name, err)
		switch {
		case errors.Is(err, overpass.ErrCityAreaNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": fmt.Sprintf("Не удалось определить границы города «%s» в OpenStreetMap", request.Name),
			})
		case errors.Is(err, overpass.ErrNoStations):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": fmt.Sprintf("Для города «%s» по заданным тегам станции не найдены", request.Name),
			})
		case errors.Is(err, context.DeadlineExceeded):
			writeJSON(w, http.StatusGatewayTimeout, map[string]string{
				"error": "OpenStreetMap отвечает слишком долго. Попробуйте ещё раз позже",
			})
		default:
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error": "Сервис OpenStreetMap временно недоступен. Попробуйте ещё раз позже",
			})
		}
		return
	}
	if err := content.Validate(); err != nil {
		log.Printf("invalid preview for city %q: %v", request.Name, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "OpenStreetMap вернул некорректную схему",
		})
		return
	}
	if len(content.Stations) == 0 || len(content.Lines) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": fmt.Sprintf("Для города «%s» не удалось построить непустую схему метро", request.Name),
		})
		return
	}

	slug, err := app.uniqueCitySlug(ctx, request.Name)
	if err != nil {
		log.Printf("create slug for city %q: %v", request.Name, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Не удалось подготовить slug города",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":          request.Name,
		"slug":          slug,
		"stationsCount": len(content.Stations),
		"linesCount":    len(content.Lines),
		"schema":        content,
	})
}

func (app application) uniqueCitySlug(ctx context.Context, name string) (string, error) {
	base := citySlug(name)
	if base == "" {
		return "", errors.New("название города нельзя преобразовать в slug")
	}
	for suffix := 1; suffix <= 1000; suffix++ {
		slug := base
		if suffix > 1 {
			slug = fmt.Sprintf("%s-%d", base, suffix)
		}
		exists, err := app.repo.CityExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
	}
	return "", errors.New("не удалось подобрать свободный slug")
}

var cityTransliteration = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func citySlug(name string) string {
	var slug strings.Builder
	pendingSeparator := false
	for _, char := range strings.ToLower(strings.TrimSpace(name)) {
		if transliteration, ok := cityTransliteration[char]; ok {
			if transliteration == "" {
				continue
			}
			if pendingSeparator && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteString(transliteration)
			pendingSeparator = false
			continue
		}
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if pendingSeparator && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteRune(char)
			pendingSeparator = false
			continue
		}
		pendingSeparator = true
	}
	return strings.Trim(slug.String(), "-")
}
