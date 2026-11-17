package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

type fakeRepository struct {
	err        error
	slug       string
	cities     []domain.City
	saved      []domain.MetroSchema
	listErr    error
	saveErr    error
	exists     bool
	existsErr  error
	nameExists bool
}

func (f *fakeRepository) ListCities(context.Context) ([]domain.City, error) {
	return f.cities, f.listErr
}
func (f *fakeRepository) GetSchemaBySlug(_ context.Context, slug string) (domain.MetroSchema, error) {
	f.slug = slug
	return domain.MetroSchema{City: domain.City{ID: 1, Name: "City", Slug: slug}, Stations: []domain.Station{{ID: 1, Year: 1982}}, Lines: []domain.MetroLine{}}, f.err
}
func (f *fakeRepository) SaveSchema(_ context.Context, schema domain.MetroSchema) error {
	f.saved = append(f.saved, schema)
	return f.saveErr
}
func (f *fakeRepository) CityExists(context.Context, string) (bool, error) {
	return f.exists, f.existsErr
}
func (f *fakeRepository) CityExistsByName(context.Context, string) (bool, error) {
	return f.nameExists, f.existsErr
}
func TestCreateCityHTTP(t *testing.T) {
	validBody := `{"name":"Казань","slug":"kazan","stations":[{"id":1,"name":"Кремлёвская","x":20,"y":30}],"lines":[{"id":1,"name":"Центральная","color":"#123456","stations":[1]}]}`

	tests := []struct {
		name         string
		body         string
		exists       bool
		existsErr    error
		saveErr      error
		wantStatus   int
		wantSaveCall bool
	}{
		{name: "creates city", body: validBody, wantStatus: http.StatusCreated, wantSaveCall: true},
		{name: "rejects existing slug", body: validBody, exists: true, wantStatus: http.StatusConflict},
		{name: "database save failure", body: validBody, saveErr: errors.New("private details"), wantStatus: http.StatusInternalServerError, wantSaveCall: true},
		{name: "rejects malformed JSON", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "rejects unknown field", body: `{"name":"Казань","slug":"kazan","stations":[],"lines":[],"extra":true}`, wantStatus: http.StatusBadRequest},
		{name: "rejects trailing JSON", body: validBody + `{}`, wantStatus: http.StatusBadRequest},
		{name: "rejects invalid slug", body: strings.Replace(validBody, `"kazan"`, `"Kazan City"`, 1), wantStatus: http.StatusBadRequest},
		{name: "rejects invalid schema", body: `{"name":"Казань","slug":"kazan","stations":[],"lines":[{"id":1,"name":"x","color":"red","stations":[]}]}`, wantStatus: http.StatusBadRequest},
		{name: "database lookup failure", body: validBody, existsErr: errors.New("private details"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{exists: tc.exists, existsErr: tc.existsErr, saveErr: tc.saveErr}
			app := application{repo: repo}
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/cities", app.createCityHandler)
			request := httptest.NewRequest(http.MethodPost, "/api/cities", bytes.NewBufferString(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d; body=%s", response.Code, tc.wantStatus, response.Body.String())
			}
			if (len(repo.saved) > 0) != tc.wantSaveCall {
				t.Fatalf("saved schemas=%d, want save call=%v", len(repo.saved), tc.wantSaveCall)
			}
			if tc.wantStatus < http.StatusInternalServerError && tc.wantSaveCall {
				saved := repo.saved[0]
				if saved.City.Name != "Казань" || saved.City.Slug != "kazan" || len(saved.Stations) != 1 {
					t.Fatalf("unexpected saved schema: %+v", saved)
				}
			}
		})
	}
}

func TestAdminOnlyRequiresConfiguredBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		authority  string
		wantStatus int
	}{
		{name: "missing token", token: "secret", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", token: "secret", authority: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid token", token: "secret", authority: "Bearer secret", wantStatus: http.StatusNoContent},
		{name: "admin disabled", wantStatus: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := adminOnly(test.token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
			if test.authority != "" {
				request.Header.Set("Authorization", test.authority)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestPreviewCityUsesRussianNameAndGeneratedSlug(t *testing.T) {
	app := application{
		repo: &fakeRepository{},
		previewCity: func(_ context.Context, name string) (domain.MetroContent, error) {
			if name != "Казань" {
				t.Fatalf("preview city name=%q, want Казань", name)
			}
			return domain.MetroContent{
				Stations: []domain.Station{{ID: 1, Name: "Кремлёвская", X: 10, Y: 20}},
				Lines:    []domain.MetroLine{{ID: 1, Name: "Красная", Color: "#e53935", Stations: []int{1}}},
			}, nil
		},
	}
	mux := http.NewServeMux()
	mux.Handle("POST /api/admin/cities/preview", adminOnly("secret", http.HandlerFunc(app.previewCityHandler)))
	request := httptest.NewRequest(http.MethodPost, "/api/admin/cities/preview", strings.NewReader(`{"name":"Казань"}`))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Slug          string `json:"slug"`
		StationsCount int    `json:"stationsCount"`
		LinesCount    int    `json:"linesCount"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Slug != "kazan" || result.StationsCount != 1 || result.LinesCount != 1 {
		t.Fatalf("unexpected preview response: %+v", result)
	}
}

func TestPreviewCityRejectsDuplicateBeforeCallingOSM(t *testing.T) {
	called := false
	app := application{
		repo: &fakeRepository{nameExists: true},
		previewCity: func(context.Context, string) (domain.MetroContent, error) {
			called = true
			return domain.MetroContent{}, nil
		},
	}
	mux := http.NewServeMux()
	mux.Handle("POST /api/admin/cities/preview", adminOnly("secret", http.HandlerFunc(app.previewCityHandler)))
	request := httptest.NewRequest(http.MethodPost, "/api/admin/cities/preview", strings.NewReader(`{"name":"Казань"}`))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || called {
		t.Fatalf("status=%d called OSM=%v; want conflict without OSM request", response.Code, called)
	}
	if !strings.Contains(response.Body.String(), "уже добавлен") {
		t.Fatalf("duplicate response=%s", response.Body.String())
	}
}

func TestCitySlugTransliteration(t *testing.T) {
	if got := citySlug("Нижний Новгород"); got != "nizhniy-novgorod" {
		t.Fatalf("citySlug=%q, want nizhniy-novgorod", got)
	}
}

func TestPreviewImportHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/cities/preview-import", (application{}).previewImportHandler)

	t.Run("valid Overpass upload", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("file", "stations.json")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(`{"elements":[{"type":"node","id":1,"lat":59.9,"lon":30.3,"tags":{"station":"subway","name":"Station A","colour":"blue"}},{"type":"node","id":2,"lat":59.91,"lon":30.31,"tags":{"station":"subway","name":"Station B","colour":"blue"}},{"type":"relation","id":10,"tags":{"type":"route","route":"subway","ref":"2","colour":"blue"},"members":[{"type":"node","ref":1,"role":"stop"},{"type":"node","ref":2,"role":"stop"}]}]}`))
		_ = writer.Close()
		request := httptest.NewRequest(http.MethodPost, "/api/cities/preview-import", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		var result struct {
			StationsCount int `json:"stationsCount"`
			LinesCount    int `json:"linesCount"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.StationsCount != 2 || result.LinesCount != 1 {
			t.Fatalf("preview counts=%d stations, %d lines", result.StationsCount, result.LinesCount)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/cities/preview-import", nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", response.Code, http.StatusBadRequest)
		}
	})
}

func TestCreateCityRejectsExistingSlug(t *testing.T) {
	repo := &fakeRepository{exists: true}
	app := application{repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/cities", app.createCityHandler)
	body := `{"name":"Казань","slug":"kazan","stations":[{"id":1,"name":"Кремлёвская","x":20,"y":30}],"lines":[{"id":1,"name":"Центральная","color":"#123456","stations":[1]}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/cities", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || len(repo.saved) != 0 {
		t.Fatalf("status=%d saved=%d, want conflict and no save", response.Code, len(repo.saved))
	}
}

func TestCreateCityRejectsOversizedBody(t *testing.T) {
	repo := &fakeRepository{}
	app := application{repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/cities", app.createCityHandler)
	request := httptest.NewRequest(http.MethodPost, "/api/cities", bytes.NewBufferString(strings.Repeat(" ", (2<<20)+1)))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestSchemaHTTP(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"success", nil, 200}, {"missing", repository.ErrNotFound, 404}, {"database failure", errors.New("private database details"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{err: tc.err}
			app := application{repo: repo}
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/cities/{citySlug}/metro/schema", app.schemaHandler)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/cities/example/metro/schema", nil))
			if w.Code != tc.status || repo.slug != "example" {
				t.Fatalf("status=%d slug=%s", w.Code, repo.slug)
			}
			if tc.err == nil {
				var schema domain.MetroSchema
				if err := json.Unmarshal(w.Body.Bytes(), &schema); err != nil {
					t.Fatal(err)
				}
				if schema.City.Slug != "example" || schema.Stations[0].Year != 1982 || schema.Lines == nil {
					t.Fatal("response contract changed")
				}
			} else {
				var body map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["error"] == "" || body["error"] == tc.err.Error() {
					t.Fatal("invalid public error")
				}
			}
		})
	}
}
