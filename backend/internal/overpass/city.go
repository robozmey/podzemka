package overpass

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
)

const (
	nominatimSearchURL = "https://nominatim.openstreetmap.org/search"
	maxImportResponse  = 10 << 20
	geocoderAttempt    = 6 * time.Second
	overpassAttempt    = 95 * time.Second
	overpassFallback   = 25 * time.Second
)

var (
	ErrCityAreaNotFound = errors.New("не найдена административная граница города")
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type placeResult struct {
	OSMType  string `json:"osm_type"`
	OSMID    int64  `json:"osm_id"`
	Category string `json:"category"`
	Type     string `json:"type"`
	Address  struct {
		Country string `json:"country"`
	} `json:"address"`
}

// FetchCity finds a city's administrative area in OSM and imports its subway stations.
func FetchCity(ctx context.Context, name string) (domain.MetroContent, error) {
	client := &http.Client{Timeout: 97 * time.Second}
	return fetchCityWithFallbacks(ctx, name, client,
		serviceEndpoints("NOMINATIM_URL", nominatimSearchURL, "https://nominatim.openstreetmap.fr/search"),
		serviceEndpoints("OVERPASS_URL", "https://overpass-api.de/api/interpreter", "https://overpass.kumi.systems/api/interpreter", "https://overpass.private.coffee/api/interpreter"),
	)
}

func fetchCity(ctx context.Context, name string, client httpDoer, nominatimURL, overpassURL string) (domain.MetroContent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.MetroContent{}, errors.New("укажите название города")
	}

	areaID, err := findCityArea(ctx, client, nominatimURL, name)
	if err != nil {
		return domain.MetroContent{}, err
	}
	return fetchArea(ctx, name, client, areaID, overpassURL)
}

func fetchCityWithFallbacks(ctx context.Context, name string, client httpDoer, nominatimURLs, overpassURLs []string) (domain.MetroContent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.MetroContent{}, errors.New("укажите название города")
	}

	var areaID int64
	var lastError error
	for index, endpoint := range nominatimURLs {
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, geocoderAttempt)
		areaID, lastError = findCityArea(attemptCtx, client, endpoint, name)
		cancel()
		log.Printf("OSM geocoder attempt %d/%d took %s: %v", index+1, len(nominatimURLs), time.Since(started).Round(time.Millisecond), lastError)
		if lastError == nil {
			break
		}
		if ctx.Err() != nil {
			return domain.MetroContent{}, fmt.Errorf("Nominatim geocoding: %w (last error: %v)", ctx.Err(), lastError)
		}
	}
	if lastError != nil {
		return domain.MetroContent{}, fmt.Errorf("Nominatim endpoints недоступны: %w", lastError)
	}

	for index, endpoint := range overpassURLs {
		attemptTimeout := overpassFallback
		if index == 0 {
			attemptTimeout = overpassAttempt
		}
		started := time.Now()
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		content, err := fetchArea(attemptCtx, name, client, areaID, endpoint)
		cancel()
		log.Printf("OSM Overpass attempt %d/%d took %s: %v", index+1, len(overpassURLs), time.Since(started).Round(time.Millisecond), err)
		if err == nil {
			return content, nil
		}
		lastError = err
		if ctx.Err() != nil {
			return domain.MetroContent{}, fmt.Errorf("Overpass stage: %w (last error: %v)", ctx.Err(), lastError)
		}
	}
	return domain.MetroContent{}, fmt.Errorf("все Overpass endpoints недоступны или не вернули маршруты: %w", lastError)
}

func serviceEndpoints(variable string, defaults ...string) []string {
	configured := strings.Split(os.Getenv(variable), ",")
	endpoints := make([]string, 0, len(configured)+len(defaults))
	seen := make(map[string]bool)
	for _, endpoint := range append(configured, defaults...) {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" || seen[endpoint] {
			continue
		}
		seen[endpoint] = true
		endpoints = append(endpoints, endpoint)
	}
	return endpoints
}

func fetchArea(ctx context.Context, _ string, client httpDoer, areaID int64, overpassURL string) (domain.MetroContent, error) {
	data, err := postOverpassQuery(ctx, client, overpassURL, stationsQuery(areaID))
	if err != nil {
		return domain.MetroContent{}, fmt.Errorf("запрос станций Overpass: %w", err)
	}
	schema, err := Parse(strings.NewReader(string(data)), MapWidth, MapHeight)
	if err != nil {
		return domain.MetroContent{}, fmt.Errorf("разобрать станции города: %w", err)
	}
	if err := schema.Validate(); err != nil {
		return domain.MetroContent{}, fmt.Errorf("проверить схему города: %w", err)
	}
	return schema, nil
}

func postOverpassQuery(ctx context.Context, client httpDoer, overpassURL, query string) ([]byte, error) {
	form := url.Values{"data": {query}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, overpassURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("подготовить запрос Overpass: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", osmUserAgent())

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("запрос Overpass: %w", err)
	}
	defer response.Body.Close()
	return readAPIResponse(response)
}

func findCityArea(ctx context.Context, client httpDoer, endpoint, name string) (int64, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return 0, fmt.Errorf("адрес Nominatim некорректен: %w", err)
	}
	query := parsedURL.Query()
	query.Set("q", strings.TrimSpace(name))
	query.Set("countrycodes", "ru")
	query.Set("format", "jsonv2")
	query.Set("addressdetails", "1")
	query.Set("limit", "20")
	parsedURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("подготовить запрос Nominatim: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Language", "ru")
	request.Header.Set("User-Agent", osmUserAgent())

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("запрос Nominatim: %w", err)
	}
	defer response.Body.Close()
	data, err := readAPIResponse(response)
	if err != nil {
		return 0, fmt.Errorf("Nominatim: %w", err)
	}

	var places []placeResult
	if err := json.Unmarshal(data, &places); err != nil {
		return 0, fmt.Errorf("разобрать ответ Nominatim: %w", err)
	}
	for _, place := range places {
		isAdministrativeBoundary := place.Category == "boundary" && place.Type == "administrative"
		isNamedCityArea := place.Category == "place" && (place.Type == "city" || place.Type == "town")
		if (!isAdministrativeBoundary && !isNamedCityArea) || place.OSMID <= 0 {
			continue
		}
		if place.Address.Country != "" && !strings.EqualFold(place.Address.Country, "Россия") && !strings.EqualFold(place.Address.Country, "Russia") {
			continue
		}
		switch place.OSMType {
		case "relation":
			return 3_600_000_000 + place.OSMID, nil
		case "way":
			return 2_400_000_000 + place.OSMID, nil
		}
	}
	return 0, fmt.Errorf("%w: «%s» в России", ErrCityAreaNotFound, name)
}

func stationsQuery(areaID int64) string {
	return fmt.Sprintf(`[out:json][timeout:60];
area(%d)->.city;
node
	["station"="subway"]
	["railway"="station"]
	["subway"="yes"]
	["construction"!~".*"]
	["proposed"!~".*"]
	["disused"!~".*"]
	["abandoned"!~".*"]
	(area.city);
out body;`, areaID)
}

func osmUserAgent() string {
	if email := strings.TrimSpace(os.Getenv("OSM_CONTACT_EMAIL")); email != "" {
		return "Podzemka city importer (" + email + ")"
	}
	return "Podzemka city importer"
}

func readAPIResponse(response *http.Response) ([]byte, error) {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("сервер вернул HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxImportResponse+1))
	if err != nil {
		return nil, fmt.Errorf("прочитать ответ сервера: %w", err)
	}
	if len(data) > maxImportResponse {
		return nil, errors.New("ответ сервера превышает 10 МБ")
	}
	return data, nil
}
