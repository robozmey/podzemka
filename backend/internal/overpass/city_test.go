package overpass

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchCityUsesOSMAreaAndKeepsCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			if r.URL.Query().Get("q") != "Казань" {
				t.Errorf("Nominatim query = %q", r.URL.Query().Get("q"))
			}
			if r.URL.Query().Get("countrycodes") != "ru" {
				t.Errorf("countrycodes = %q", r.URL.Query().Get("countrycodes"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"osm_type":"relation","osm_id":1234,"category":"boundary","type":"administrative","address":{"country":"Россия"}}]`)
		case "/interpreter":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse Overpass form: %v", err)
			}
			query := r.Form.Get("data")
			w.Header().Set("Content-Type", "application/json")
			for _, required := range []string{
				"area(3600001234)",
				"[out:json][timeout:60]",
				`["station"="subway"]`,
				`["railway"="station"]`,
				`["subway"="yes"]`,
				`["construction"!~".*"]`,
				`["proposed"!~".*"]`,
				`["disused"!~".*"]`,
				`["abandoned"!~".*"]`,
			} {
				if !strings.Contains(query, required) {
					t.Errorf("station query missing %q", required)
				}
			}
			_, _ = io.WriteString(w, `{"elements":[{"type":"node","id":1,"lat":55.75,"lon":49.12,"tags":{"station":"subway","railway":"station","subway":"yes","name":"Кремлёвская","colour":"red"}},{"type":"node","id":2,"lat":55.751,"lon":49.121,"tags":{"station":"subway","railway":"station","subway":"yes","name":"Площадь Тукая","colour":"red"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	content, err := fetchCity(context.Background(), "Казань", server.Client(), server.URL+"/search", server.URL+"/interpreter")
	if err != nil {
		t.Fatalf("fetchCity: %v", err)
	}
	if len(content.Stations) != 2 || len(content.Lines) != 1 {
		t.Fatalf("unexpected result: %d stations, %d lines", len(content.Stations), len(content.Lines))
	}
	station := content.Stations[0]
	if station.Name != "Кремлёвская" || station.Latitude == nil || *station.Latitude != 55.75 || station.Longitude == nil || *station.Longitude != 49.12 {
		t.Fatalf("unexpected station: %+v", station)
	}
	if got := content.Lines[0].Stations; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("station order=%v, want [1 2] from nearest-neighbor order", got)
	}
}

func TestFindCityAreaAcceptsNominatimCityResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"osm_type":"relation","osm_id":987,"category":"place","type":"city","address":{"country":"Россия"}}]`)
	}))
	defer server.Close()

	areaID, err := findCityArea(context.Background(), server.Client(), server.URL, "Новосибирск")
	if err != nil {
		t.Fatalf("findCityArea: %v", err)
	}
	if areaID != 3_600_000_987 {
		t.Fatalf("area ID=%d, want %d", areaID, 3_600_000_987)
	}
}

func TestFetchCityFallsBackToNextOSMEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nominatim-primary", "/overpass-primary":
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		case "/nominatim-secondary":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"osm_type":"relation","osm_id":42,"category":"boundary","type":"administrative","address":{"country":"Россия"}}]`)
		case "/overpass-secondary":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"elements":[{"type":"node","id":1,"lat":55.75,"lon":49.12,"tags":{"station":"subway","railway":"station","subway":"yes","name":"A","colour":"red"}},{"type":"node","id":2,"lat":55.76,"lon":49.13,"tags":{"station":"subway","railway":"station","subway":"yes","name":"B","colour":"red"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	content, err := fetchCityWithFallbacks(
		context.Background(),
		"Казань",
		server.Client(),
		[]string{server.URL + "/nominatim-primary", server.URL + "/nominatim-secondary"},
		[]string{server.URL + "/overpass-primary", server.URL + "/overpass-secondary"},
	)
	if err != nil {
		t.Fatalf("fetchCityWithFallbacks: %v", err)
	}
	if len(content.Stations) != 2 || len(content.Lines) != 1 {
		t.Fatalf("fallback result: %d stations, %d lines", len(content.Stations), len(content.Lines))
	}
}
