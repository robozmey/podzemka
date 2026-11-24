package overpass

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func stationNode(id int64, name string, lon float64, year string) element {
	return element{Type: "node", ID: id, Lat: 55, Lon: lon, Tags: map[string]string{
		"railway": "station", "station": "subway", "name": name, "start_date": year,
	}}
}

func stopNode(id int64, name string, lon float64) element {
	return element{Type: "node", ID: id, Lat: 55, Lon: lon, Tags: map[string]string{
		"railway": "stop", "public_transport": "stop_position", "subway": "yes", "name": name,
	}}
}

func subwayRoute(id int64, ref string, stops ...int64) element {
	route := element{Type: "relation", ID: id, Tags: map[string]string{
		"route": "subway", "ref": ref, "name": "Line " + ref, "colour": "blue",
	}}
	for _, id := range stops {
		route.Members = append(route.Members, relationMember{Type: "node", Ref: id, Role: "stop"})
	}
	return route
}

func stopArea(id int64, members ...int64) element {
	area := element{Type: "relation", ID: id, Tags: map[string]string{
		"type": "public_transport", "public_transport": "stop_area",
	}}
	for _, id := range members {
		area.Members = append(area.Members, relationMember{Type: "node", Ref: id})
	}
	return area
}

func TestParseMapsBothDirectionsToStationMetadata(t *testing.T) {
	elements := []element{
		stationNode(1, "A", 37, "1980-01-01"), stationNode(2, "B", 37.01, "1990"),
		stopNode(11, "A", 37.0001), stopNode(12, "A", 36.9999),
		stopNode(21, "B", 37.0101), stopNode(22, "B", 37.0099),
		subwayRoute(90, "1", 11, 21), subwayRoute(91, "1", 22, 12),
	}
	raw, err := json.Marshal(response{Elements: elements})
	if err != nil {
		t.Fatal(err)
	}
	schema, err := Parse(strings.NewReader(string(raw)), MapWidth, MapHeight)
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(schema.Stations) != 2 || len(schema.Lines) != 1 || !reflect.DeepEqual(schema.Lines[0].Stations, []int{1, 2}) {
		t.Fatalf("unexpected canonical schema: %+v", schema)
	}
	for i, wantYear := range []int{1980, 1990} {
		station := schema.Stations[i]
		if station.Year != wantYear || *station.Longitude != elements[i].Lon || *station.Latitude != elements[i].Lat {
			t.Errorf("metadata came from a stop instead of station: %+v", station)
		}
	}
	if schema.Lines[0].Color != "#1976d2" {
		t.Fatalf("route color lost: %+v", schema.Lines[0])
	}
	if !reflect.DeepEqual(schema.Lines[0].Edges, [][2]int{{1, 2}}) {
		t.Fatalf("opposite direction stops must produce one edge: %v", schema.Lines[0].Edges)
	}
}

func TestStationResolverMatching(t *testing.T) {
	for _, tc := range []struct {
		name     string
		elements []element
		routeID  int64
		stopID   int64
		wantID   int64 // zero means the match must be rejected
	}{
		{
			name: "stop area takes precedence over name and proximity",
			elements: []element{
				stationNode(1, "Canonical", 37, "1980"), stationNode(2, "Stop name", 37.01, "1990"),
				stopNode(11, "Stop name", 37.01), stopArea(100, 1, 11), subwayRoute(90, "1", 11),
			}, routeID: 90, stopID: 11, wantID: 1,
		},
		{
			name: "same-name nearby interchange stations remain ambiguous",
			elements: []element{
				stationNode(1, "A", 37, "1980"), stationNode(2, "A", 37.001, "1990"),
				stopNode(11, "A", 37.0001), subwayRoute(90, "1", 11),
			}, routeID: 90, stopID: 11,
		},
		{
			name:     "same name outside matching radius is insufficient",
			elements: []element{stationNode(1, "A", 37, "1980"), stopNode(11, "A", 37.1), subwayRoute(90, "1", 11)},
			routeID:  90, stopID: 11,
		},
		{
			name: "explicit other direction disambiguates within the line",
			elements: []element{
				stationNode(1, "A", 37, "1980"), stationNode(2, "A", 37.001, "1990"), stopNode(11, "A", 37.0005),
				subwayRoute(90, "1", 11), subwayRoute(91, "1", 1), subwayRoute(92, "2", 2),
			}, routeID: 90, stopID: 11, wantID: 1,
		},
		{
			name: "stop areas for two lines must not be merged by name",
			elements: []element{
				stationNode(1, "A", 37, "1980"), stationNode(2, "A", 37.001, "1990"),
				stopNode(11, "A", 37.0001), stopNode(12, "A", 37.0009),
				stopArea(100, 1, 11), stopArea(101, 2, 12), subwayRoute(90, "1", 11), subwayRoute(91, "2", 12),
			}, routeID: 91, stopID: 12, wantID: 2,
		},
		{
			name: "broad interchange stop area without line evidence is ambiguous",
			elements: []element{
				stationNode(1, "A", 37, "1980"), stationNode(2, "A", 37.001, "1990"),
				stopNode(11, "A", 37.0005), stopArea(100, 1, 2, 11), subwayRoute(90, "1", 11),
			}, routeID: 90, stopID: 11,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stations := parseStations(tc.elements)
			resolver := newStationResolver(stations, tc.elements)
			station, found := resolver.resolve(tc.routeID, tc.stopID)
			if found != (tc.wantID != 0) || station.osmID != tc.wantID {
				t.Fatalf("match=(%d,%v), want station %d", station.osmID, found, tc.wantID)
			}
			if _, cached := resolver.matches[routeStopKey{routeID: tc.routeID, nodeID: tc.stopID}]; !cached {
				t.Fatal("resolved alias was not saved")
			}
		})
	}
}

func TestStationResolverScopesAliasesToLine(t *testing.T) {
	elements := []element{
		stationNode(1, "A", 37, "1980"), stationNode(2, "A", 37.001, "1990"), stopNode(11, "A", 37.0005),
		subwayRoute(90, "1", 11, 1), subwayRoute(91, "2", 11, 2),
	}
	resolver := newStationResolver(parseStations(elements), elements)
	for _, pair := range [][2]int64{{90, 1}, {91, 2}} {
		station, found := resolver.resolve(pair[0], 11)
		if !found || station.osmID != pair[1] {
			t.Fatalf("route %d inherited a different line's station: %+v", pair[0], station)
		}
	}
}

func TestParseStationsExcludesStopsAndPlatforms(t *testing.T) {
	platform := stopNode(12, "A", 37)
	platform.Tags["railway"] = "platform"
	platform.Tags["public_transport"] = "platform"
	stop := stopNode(11, "A", 37)
	// Even station=subway on a stop must not make it a passenger station.
	stop.Tags["station"] = "subway"
	stations := parseStations([]element{stationNode(1, "A", 37, "1980"), stop, platform})
	if len(stations) != 1 || stations[0].osmID != 1 {
		t.Fatalf("stop positions leaked into stations: %+v", stations)
	}
}
