package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/robozmey/podzemka/backend/internal/domain"
	"github.com/robozmey/podzemka/backend/internal/overpass"
	"github.com/robozmey/podzemka/backend/internal/repository"
)

type schemaRepository interface {
	ListCities(context.Context) ([]domain.City, error)
	GetSchemaBySlug(context.Context, string) (domain.MetroSchema, error)
	SaveSchema(context.Context, domain.MetroSchema) error
	CityExists(context.Context, string) (bool, error)
	CityExistsByName(context.Context, string) (bool, error)
}

type application struct {
	repo        schemaRepository
	previewCity func(context.Context, string) (domain.MetroContent, error)
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("Ошибка загрузки .env: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := repository.OpenPostgres(ctx)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := application{
		repo:        repository.NewPostgresRepository(db),
		previewCity: overpass.FetchCity,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", helloHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/cities", app.citiesHandler)
	mux.HandleFunc("GET /api/catalog/cities", catalogCitiesHandler)
	adminToken := os.Getenv("ADMIN_TOKEN")
	mux.Handle("GET /api/admin/session", adminOnly(adminToken, http.HandlerFunc(adminSessionHandler)))
	mux.Handle("POST /api/admin/cities/preview", adminOnly(adminToken, http.HandlerFunc(app.previewCityHandler)))
	mux.Handle("POST /api/cities", adminOnly(adminToken, http.HandlerFunc(app.createCityHandler)))
	mux.Handle("POST /api/cities/preview-import", adminOnly(adminToken, http.HandlerFunc(app.previewImportHandler)))
	mux.HandleFunc("GET /api/cities/{citySlug}/metro/schema", app.schemaHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("API is running on http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}
