package overpass

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseLineCompositionFromRelations(t *testing.T) {
	route := func(id int64, ref, name string, stops ...int64) element {
		r := element{Type: "relation", ID: id, Tags: map[string]string{
			"route": "subway", "ref": ref, "name": name, "colour": "blue",
		}}
		for _, stop := range stops {
			r.Members = append(r.Members, relationMember{Type: "node", Ref: stop, Role: "stop"})
		}
		return r
	}
	for _, tc := range []struct {
		name   string
		routes []element
		want   [][]string
		names  []string
		colors []string
	}{
		{
			name:   "uncolored stations included and unrelated colored station excluded",
			routes: []element{route(90, "1", "First", 3, 1, 2)},
			want:   [][]string{{"C", "A", "B"}}, names: []string{"First"}, colors: []string{"#1976d2"},
		},
		{
			name:   "directions with equal ref merge",
			routes: []element{route(90, "1", "First", 3, 1, 2), route(91, "1", "Reverse", 2, 1, 3)},
			want:   [][]string{{"C", "A", "B"}}, names: []string{"First"}, colors: []string{"#1976d2"},
		},
		{
			name:   "equal colors do not merge different refs",
			routes: []element{route(90, "1", "First", 3, 1), route(91, "2", "Second", 2, 4)},
			want:   [][]string{{"C", "A"}, {"B", "D"}}, names: []string{"First", "Second"}, colors: []string{"#1976d2", "#1976d2"},
		},
		{
			name:   "missing refs do not merge by color",
			routes: []element{route(90, "", "First", 3, 1), route(91, "", "Second", 2, 4)},
			want:   [][]string{{"C", "A"}, {"B", "D"}}, names: []string{"First", "Second"}, colors: []string{"#1976d2", "#1976d2"},
		},
		{
			name: "master groups different refs and supplies metadata",
			routes: []element{route(90, "1a", "Forward", 3, 1, 2), route(91, "1b", "Reverse", 2, 1, 3), {
				Type: "relation", ID: 100,
				Tags:    map[string]string{"type": "route_master", "route_master": "subway", "name": "Master", "colour": "green"},
				Members: []relationMember{{Type: "relation", Ref: 90}, {Type: "relation", Ref: 91}},
			}},
			want: [][]string{{"C", "A", "B"}}, names: []string{"Master"}, colors: []string{"#2e9d50"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			elements := make([]element, 0)
			for i, name := range []string{"A", "B", "C", "D"} {
				tags := map[string]string{"station": "subway", "name": name}
				if name == "D" {
					tags["colour"] = "blue"
				}
				elements = append(elements, element{Type: "node", ID: int64(i + 1), Lat: 55, Lon: 37 + float64(i)*0.01, Tags: tags})
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
			if len(schema.Lines) != len(tc.want) {
				t.Fatalf("lines=%d, want %d", len(schema.Lines), len(tc.want))
			}
			namesByID := make(map[int]string)
			for _, station := range schema.Stations {
				namesByID[station.ID] = station.Name
			}
			for i, line := range schema.Lines {
				var names []string
				for _, id := range line.Stations {
					names = append(names, namesByID[id])
				}
				if !reflect.DeepEqual(names, tc.want[i]) || line.Name != tc.names[i] || line.Color != tc.colors[i] {
					t.Errorf("line %d: %q %s %v; want %q %s %v", i, line.Name, line.Color, names, tc.names[i], tc.colors[i], tc.want[i])
				}
			}
		})
	}
}
