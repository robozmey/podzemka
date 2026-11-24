package overpass

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParseBuildsEdgesFromEachRoute(t *testing.T) {
	master := element{
		Type: "relation", ID: 1000,
		Tags: map[string]string{"type": "route_master", "route_master": "subway", "name": "Fork", "colour": "blue"},
		Members: []relationMember{
			{Type: "relation", Ref: 90}, {Type: "relation", Ref: 91},
			{Type: "relation", Ref: 92}, {Type: "relation", Ref: 93},
		},
	}
	withPlatform := subwayRoute(90, "1", 10, 30)
	withPlatform.Members = []relationMember{
		{Type: "node", Ref: 10, Role: "stop_entry_only"},
		{Type: "node", Ref: 999, Role: "platform"},
		{Type: "way", Ref: 888},
		{Type: "node", Ref: 30, Role: "stop_exit_only"},
	}
	for _, tc := range []struct {
		name         string
		routes       []element
		wantEdges    []string
		wantStations []string
	}{
		{
			name:      "route order overrides geographic proximity",
			routes:    []element{subwayRoute(90, "1", 20, 10, 30)},
			wantEdges: []string{"A-B", "A-C"}, wantStations: []string{"A", "B", "C"},
		},
		{
			name:      "opposite directions close a ring without duplicate edges",
			routes:    []element{subwayRoute(90, "1", 10, 30, 20, 10), subwayRoute(91, "1", 10, 20, 30, 10)},
			wantEdges: []string{"A-B", "A-C", "B-C"}, wantStations: []string{"A", "B", "C"},
		},
		{
			name: "branches share a trunk without connecting their ends",
			routes: []element{
				subwayRoute(90, "4", 10, 30, 20), subwayRoute(91, "4", 20, 30, 10),
				subwayRoute(92, "4A", 10, 30, 40), subwayRoute(93, "4A", 40, 30, 10), master,
			},
			wantEdges: []string{"A-B", "B-C", "B-D"}, wantStations: []string{"A", "B", "C", "D"},
		},
		{
			name:      "unresolved stop breaks connectivity but retains isolated station",
			routes:    []element{subwayRoute(90, "1", 10, 999, 30, 20)},
			wantEdges: []string{"B-C"}, wantStations: []string{"A", "B", "C"},
		},
		{
			name:      "separate singleton routes do not invent an edge",
			routes:    []element{subwayRoute(90, "1", 10), subwayRoute(91, "1", 30)},
			wantEdges: []string{}, wantStations: []string{"A", "B"},
		},
		{
			name:      "consecutive aliases produce no self edge",
			routes:    []element{stopNode(11, "A", 37.0001), stopNode(12, "A", 36.9999), subwayRoute(90, "1", 11, 12, 30)},
			wantEdges: []string{"A-B"}, wantStations: []string{"A", "B"},
		},
		{
			name:      "nonconsecutive revisits are retained when building edges",
			routes:    []element{subwayRoute(90, "1", 10, 30, 20, 10, 40)},
			wantEdges: []string{"A-B", "A-C", "A-D", "B-C"}, wantStations: []string{"A", "B", "C", "D"},
		},
		{
			name:      "track and platform members do not break the stop sequence",
			routes:    []element{withPlatform},
			wantEdges: []string{"A-B"}, wantStations: []string{"A", "B"},
		},
		{
			name: "ambiguous station also breaks connectivity",
			routes: []element{
				stationNode(50, "X", 37.02, "1980"), stationNode(60, "X", 37.0201, "1990"),
				stopNode(51, "X", 37.02005), subwayRoute(90, "1", 10, 51, 30),
			},
			wantEdges: []string{}, wantStations: []string{"A", "B"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// OSM IDs intentionally differ from output IDs, including their order.
			elements := []element{
				stationNode(10, "A", 37, "1980"), stationNode(30, "B", 37.01, "1980"),
				stationNode(20, "C", 37.02, "1980"), stationNode(40, "D", 37.03, "1980"),
			}
			elements = append(elements, tc.routes...)
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
			if len(schema.Lines) != 1 {
				t.Fatalf("lines=%d, want 1", len(schema.Lines))
			}
			line := schema.Lines[0]
			if line.Edges == nil {
				t.Fatal("processed routes must emit an explicit edges array")
			}
			byID := make(map[int]string)
			for _, station := range schema.Stations {
				byID[station.ID] = station.Name
			}
			edges := make([]string, 0, len(line.Edges))
			for _, edge := range line.Edges {
				ends := []string{byID[edge[0]], byID[edge[1]]}
				sort.Strings(ends)
				edges = append(edges, strings.Join(ends, "-"))
			}
			sort.Strings(edges)
			stations := make([]string, 0, len(line.Stations))
			for _, id := range line.Stations {
				stations = append(stations, byID[id])
			}
			sort.Strings(stations)
			if !reflect.DeepEqual(edges, tc.wantEdges) || !reflect.DeepEqual(stations, tc.wantStations) {
				t.Fatalf("edges=%v stations=%v; want edges=%v stations=%v", edges, stations, tc.wantEdges, tc.wantStations)
			}
		})
	}
}

func TestParseStationOnlyExportRetainsLegacyConnectivity(t *testing.T) {
	raw := `{"elements":[{"type":"node","id":1,"lat":55,"lon":37,"tags":{"station":"subway","name":"A","colour":"blue"}}]}`
	schema, err := Parse(strings.NewReader(raw), MapWidth, MapHeight)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Lines) != 1 || schema.Lines[0].Edges != nil {
		t.Fatalf("station-only import must not claim explicit route edges: %+v", schema.Lines)
	}
}
