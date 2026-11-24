package overpass

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/robozmey/podzemka/backend/internal/domain"
)

const (
	MapWidth  = 700.0
	MapHeight = 420.0
	mapMargin = 30.0
)

var ErrNoStations = errors.New("в выгрузке не найдены узлы станций метро с названием")

type response struct {
	Elements []element `json:"elements"`
}

type rawElement = element

type element struct {
	Type    string            `json:"type"`
	ID      int64             `json:"id"`
	Lat     float64           `json:"lat"`
	Lon     float64           `json:"lon"`
	Tags    map[string]string `json:"tags"`
	Members []member          `json:"members"`
	Nodes   []int64           `json:"nodes"`
}

type member struct {
	Type string `json:"type"`
	Ref  int64  `json:"ref"`
}

type importedStation struct {
	osmID int64
	name  string
	lat   float64
	lon   float64
	year  int
	color string
}

type importedLine struct {
	key        string
	name       string
	color      string
	stationIDs []int64
}

var lineColors = []struct {
	name  string
	color string
}{
	{name: "Красная", color: "#e53935"},
	{name: "Синяя", color: "#1976d2"},
	{name: "Зелёная", color: "#2e9d50"},
	{name: "Оранжевая", color: "#f28c28"},
	{name: "Фиолетовая", color: "#8e44ad"},
	{name: "Коричневая", color: "#795548"},
}

// Parse converts a raw Overpass JSON export into a metro schema content object.
func Parse(input io.Reader, width, height float64) (domain.MetroContent, error) {
	if width <= 2*mapMargin || height <= 2*mapMargin {
		return domain.MetroContent{}, errors.New("ширина и высота должны быть больше двойного отступа")
	}
	data, err := parseElements(input)
	if err != nil {
		return domain.MetroContent{}, err
	}
	stations, lines := parseStationsAndLines(data.Elements)
	if len(stations) == 0 {
		return domain.MetroContent{}, ErrNoStations
	}
	return buildSchema(stations, lines, width, height), nil
}

func parseElements(input io.Reader) (response, error) {
	payload, err := io.ReadAll(input)
	if err != nil {
		return response{}, fmt.Errorf("прочитать Overpass JSON: %w", err)
	}

	var data response
	if err := json.Unmarshal(payload, &data); err != nil {
		// Some exports contain comma-separated nodes without the enclosing
		// {"elements": [...]} object and may have trailing commas.
		cleanPayload := removeTrailingCommas(bytes.TrimSpace(payload))
		var rawElements []rawElement
		wrapped := append(append([]byte{'['}, cleanPayload...), ']')
		if arrayErr := json.Unmarshal(wrapped, &rawElements); arrayErr != nil {
			return response{}, fmt.Errorf("разобрать Overpass JSON: %w", err)
		}
		data.Elements = rawElements
	}
	return data, nil
}

func parseStations(elements []element) []importedStation {
	stations, _ := parseStationsAndLines(elements)
	return stations
}

func parseStationsAndLines(elements []element) ([]importedStation, []importedLine) {
	unique := make(map[int64]importedStation)
	lines := make([]importedLine, 0)
	lineIndexes := make(map[string]int)
	for _, item := range elements {
		if item.Type != "relation" || (item.Tags["route"] != "subway" && item.Tags["route"] != "light_rail") {
			continue
		}
		color := normalizeColor(firstStationName(item.Tags, "colour", "color"))
		key := strings.TrimSpace(item.Tags["ref"])
		if key == "" {
			key = strings.TrimSpace(item.Tags["name"])
		}
		if key == "" {
			key = fmt.Sprintf("relation-%d", item.ID)
		}
		if _, ok := lineIndexes[key]; !ok {
			lineIndexes[key] = len(lines)
			name := firstStationName(item.Tags, "name:ru", "name:en", "alt_name:en", "official_name:en", "int_name", "name:zh-Latn", "name:zh_pinyin", "name", "ref")
			if name == "" {
				name = fmt.Sprintf("Линия %d", len(lines)+1)
			}
			if color == "#808080" {
				color = lineColors[len(lines)%len(lineColors)].color
			}
			lines = append(lines, importedLine{key: key, name: name, color: color})
		}
		lineIndex := lineIndexes[key]
		for _, relationMember := range item.Members {
			if relationMember.Type != "node" || relationMember.Ref == 0 {
				continue
			}
			lines[lineIndex].stationIDs = append(lines[lineIndex].stationIDs, relationMember.Ref)
		}
	}
	ways := make(map[int64][]int64)
	for _, item := range elements {
		if item.Type != "way" {
			continue
		}
		ways[item.ID] = append(ways[item.ID], item.Nodes...)
	}
	for _, item := range elements {
		if item.Type != "relation" || (item.Tags["route"] != "subway" && item.Tags["route"] != "light_rail") {
			continue
		}
		key := strings.TrimSpace(item.Tags["ref"])
		if key == "" {
			key = strings.TrimSpace(item.Tags["name"])
		}
		if key == "" {
			key = fmt.Sprintf("relation-%d", item.ID)
		}
		lineIndex, ok := lineIndexes[key]
		if !ok {
			continue
		}
		for _, relationMember := range item.Members {
			switch relationMember.Type {
			case "node":
				lines[lineIndex].stationIDs = append(lines[lineIndex].stationIDs, relationMember.Ref)
			case "way":
				lines[lineIndex].stationIDs = append(lines[lineIndex].stationIDs, ways[relationMember.Ref]...)
			}
		}
	}
	for _, item := range elements {
		if item.Type != "node" || item.ID == 0 || item.Lat < -90 || item.Lat > 90 || item.Lon < -180 || item.Lon > 180 {
			continue
		}
		if item.Tags["station"] != "subway" && item.Tags["subway"] != "yes" {
			continue
		}
		name := firstStationName(item.Tags,
			"name:ru",
			"name:en",
			"alt_name:en",
			"official_name:en",
			"int_name",
			"name:zh-Latn",
			"name:zh_pinyin",
			"name",
		)
		if name == "" {
			continue
		}
		unique[item.ID] = importedStation{
			osmID: item.ID,
			name:  name,
			lat:   item.Lat,
			lon:   item.Lon,
			year:  parseYear(item.Tags["start_date"]),
			color: normalizeColor(item.Tags["colour"]),
		}
	}

	stations := make([]importedStation, 0, len(unique))
	for _, station := range unique {
		stations = append(stations, station)
	}
	sort.Slice(stations, func(i, j int) bool {
		if stations[i].color != stations[j].color {
			return stations[i].color < stations[j].color
		}
		return stations[i].osmID < stations[j].osmID
	})
	for index := range lines {
		lines[index].stationIDs = uniqueInt64(lines[index].stationIDs)
	}
	return stations, lines
}

func uniqueInt64(values []int64) []int64 {
	seen := make(map[int64]bool, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func firstStationName(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if name := strings.TrimSpace(tags[key]); name != "" {
			return name
		}
	}
	return ""
}

func removeTrailingCommas(payload []byte) []byte {
	cleaned := make([]byte, 0, len(payload))
	inString := false
	escaped := false
	for i, char := range payload {
		if inString {
			cleaned = append(cleaned, char)
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' {
			inString = true
			cleaned = append(cleaned, char)
			continue
		}
		if char == ',' {
			j := i + 1
			for j < len(payload) && (payload[j] == ' ' || payload[j] == '\n' || payload[j] == '\r' || payload[j] == '\t') {
				j++
			}
			if j < len(payload) && (payload[j] == '}' || payload[j] == ']') {
				continue
			}
		}
		cleaned = append(cleaned, char)
	}
	return cleaned
}

func buildSchema(stations []importedStation, importedLines []importedLine, width, height float64) domain.MetroContent {
	minLat, maxLat := stations[0].lat, stations[0].lat
	minLon, maxLon := stations[0].lon, stations[0].lon
	for _, station := range stations[1:] {
		minLat = min(minLat, station.lat)
		maxLat = max(maxLat, station.lat)
		minLon = min(minLon, station.lon)
		maxLon = max(maxLon, station.lon)
	}
	latRange := maxLat - minLat
	lonRange := maxLon - minLon
	if latRange == 0 {
		latRange = 1
	}
	if lonRange == 0 {
		lonRange = 1
	}

	out := domain.MetroContent{
		Stations: make([]domain.Station, 0, len(stations)),
		Lines:    make([]domain.MetroLine, 0, len(lineColors)),
	}
	stationIDsByOSMID := make(map[int64]int, len(stations))
	for index, station := range stations {
		x := mapMargin + (station.lon-minLon)/lonRange*(width-2*mapMargin)
		y := mapMargin + (maxLat-station.lat)/latRange*(height-2*mapMargin)
		stationID := index + 1
		latitude, longitude := station.lat, station.lon
		out.Stations = append(out.Stations, domain.Station{
			ID: stationID, Name: station.name,
			X: int(x + 0.5), Y: int(y + 0.5), Year: station.year,
			Latitude: &latitude, Longitude: &longitude,
		})
		stationIDsByOSMID[station.osmID] = stationID
	}
	if len(importedLines) == 0 {
		fallback := importedLine{key: "fallback", name: lineName(stations[0].color, 1), color: stations[0].color}
		for _, station := range stations {
			fallback.stationIDs = append(fallback.stationIDs, station.osmID)
		}
		importedLines = []importedLine{fallback}
	}
	for _, line := range importedLines {
		colorStations := make([]importedStation, 0, len(line.stationIDs))
		for _, osmID := range line.stationIDs {
			for _, station := range stations {
				if station.osmID == osmID {
					colorStations = append(colorStations, station)
					break
				}
			}
		}
		if len(colorStations) < 2 {
			continue
		}
		orderedStations := orderLineStations(colorStations)
		if len(orderedStations) == 0 {
			continue
		}
		stationIDs := make([]int, 0, len(orderedStations))
		for _, station := range orderedStations {
			stationIDs = append(stationIDs, stationIDsByOSMID[station.osmID])
		}
		out.Lines = append(out.Lines, domain.MetroLine{
			ID: len(out.Lines) + 1, Name: line.name, Color: line.color, Stations: stationIDs,
		})
	}
	return out
}

func orderLineStations(stations []importedStation) []importedStation {
	if len(stations) < 2 {
		return append([]importedStation(nil), stations...)
	}
	remaining := append([]importedStation(nil), stations...)
	sort.Slice(remaining, func(i, j int) bool {
		if remaining[i].lon != remaining[j].lon {
			return remaining[i].lon < remaining[j].lon
		}
		if remaining[i].lat != remaining[j].lat {
			return remaining[i].lat < remaining[j].lat
		}
		return remaining[i].osmID < remaining[j].osmID
	})
	ordered := make([]importedStation, 0, len(remaining))
	current := remaining[0]
	ordered = append(ordered, current)
	remaining = remaining[1:]
	for len(remaining) > 0 {
		nearestIndex := 0
		nearestDistance := squaredGeoDistance(current, remaining[0])
		for i := 1; i < len(remaining); i++ {
			distance := squaredGeoDistance(current, remaining[i])
			if distance < nearestDistance || (distance == nearestDistance && remaining[i].osmID < remaining[nearestIndex].osmID) {
				nearestIndex = i
				nearestDistance = distance
			}
		}
		current = remaining[nearestIndex]
		ordered = append(ordered, current)
		remaining = append(remaining[:nearestIndex], remaining[nearestIndex+1:]...)
	}
	return ordered
}

func squaredGeoDistance(a, b importedStation) float64 {
	meanLat := (a.lat + b.lat) / 2 * math.Pi / 180
	dLon := (a.lon - b.lon) * math.Cos(meanLat)
	dLat := a.lat - b.lat
	return dLon*dLon + dLat*dLat
}

func parseYear(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil || year < 1800 || year > time.Now().Year()+10 {
		return 0
	}
	return year
}

func normalizeColor(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	colors := map[string]string{
		"red": "#e53935", "#ff0000": "#e53935", "#e31e24": "#e53935",
		"blue": "#1976d2", "#0000ff": "#1976d2", "#0078bf": "#1976d2",
		"green": "#2e9d50", "#008000": "#2e9d50", "#009a49": "#2e9d50",
		"orange": "#f28c28", "#ffa500": "#f28c28", "#f58220": "#f28c28",
		"purple": "#8e44ad", "#800080": "#8e44ad", "#702082": "#8e44ad",
		"brown": "#795548", "#a0522d": "#795548", "#8d5b2d": "#795548",
	}
	if color, ok := colors[value]; ok {
		return color
	}
	if len(value) == 7 && value[0] == '#' {
		if _, err := strconv.ParseUint(value[1:], 16, 32); err == nil {
			return value
		}
	}
	return "#808080"
}

func lineName(color string, number int) string {
	for _, line := range lineColors {
		if line.color == color {
			return line.name
		}
	}
	return fmt.Sprintf("Линия %d", number)
}
