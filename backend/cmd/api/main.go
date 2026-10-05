package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type city struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type station struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

type metroLine struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Stations []int  `json:"stations"`
}

type metroSchema struct {
	City     city        `json:"city"`
	Stations []station   `json:"stations"`
	Lines    []metroLine `json:"lines"`
}

var cities = []city{
	{ID: 1, Name: "Санкт-Петербург", Slug: "saint-petersburg"},
}

var schemas = map[string]metroSchema{
	"saint-petersburg": {
		City: cities[0],
		Stations: []station{
			{ID: 1, Name: "Вокзальная", X: 90, Y: 210},
			{ID: 2, Name: "Площадь Мира", X: 220, Y: 210},
			{ID: 3, Name: "Невская", X: 350, Y: 210},
			{ID: 4, Name: "Петроградская", X: 480, Y: 210},
			{ID: 5, Name: "Северная", X: 610, Y: 210},
			{ID: 6, Name: "Морская", X: 350, Y: 85},
			{ID: 7, Name: "Адмиралтейская", X: 350, Y: 335},
			{ID: 8, Name: "Набережная", X: 510, Y: 335},
		},
		Lines: []metroLine{
			{ID: 1, Name: "Красная", Color: "#ef4444", Stations: []int{1, 2, 3, 4, 5}},
			{ID: 2, Name: "Синяя", Color: "#3b82f6", Stations: []int{6, 3, 7, 8}},
		},
	},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", helloHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/cities", citiesHandler)
	mux.HandleFunc("GET /api/cities/{citySlug}/metro/schema", schemaHandler)

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

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("Hello, World!"))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func citiesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, cities)
}

func schemaHandler(w http.ResponseWriter, r *http.Request) {
	schema, ok := schemas[r.PathValue("citySlug")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Схема города не найдена"})
		return
	}

	writeJSON(w, http.StatusOK, schema)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
