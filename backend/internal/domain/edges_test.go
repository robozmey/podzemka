package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMetroLineEdgesJSONRoundTrip(t *testing.T) {
	line := MetroLine{ID: 1, Name: "Line", Color: "#123abc", Stations: []int{1, 2, 3}, Edges: [][2]int{{1, 2}, {2, 3}, {3, 1}}}
	raw, err := json.Marshal(line)
	if err != nil {
		t.Fatal(err)
	}
	var decoded MetroLine
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(line, decoded) {
		t.Fatalf("edges lost on JSON round trip: %+v", decoded)
	}
	var legacy MetroLine
	if err := json.Unmarshal([]byte(`{"id":1,"name":"Line","color":"#123abc","stations":[1,2]}`), &legacy); err != nil || legacy.Edges != nil {
		t.Fatalf("legacy schema must remain readable: %+v, %v", legacy, err)
	}
}

func TestMetroContentValidatesEdges(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edges [][2]int
		valid bool
	}{
		{"legacy without edges", nil, true},
		{"explicit empty edges", [][2]int{}, true},
		{"cycle", [][2]int{{1, 2}, {2, 3}, {3, 1}}, true},
		{"missing station", [][2]int{{1, 99}}, false},
		{"station outside line", [][2]int{{1, 4}}, false},
		{"self edge", [][2]int{{1, 1}}, false},
		{"duplicate undirected edge", [][2]int{{1, 2}, {2, 1}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := MetroContent{
				Stations: []Station{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}, {ID: 3, Name: "C"}, {ID: 4, Name: "D"}},
				Lines:    []MetroLine{{ID: 1, Name: "Line", Color: "#123abc", Stations: []int{1, 2, 3}, Edges: tc.edges}},
			}
			if err := content.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate()=%v, want valid=%v", err, tc.valid)
			}
		})
	}
}
