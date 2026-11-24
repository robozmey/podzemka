package overpass

import (
	"strings"
	"testing"
)

func TestParseUsesRouteRelationOrder(t *testing.T) {
	raw := `{"elements":[
		{"type":"node","id":1,"lat":55,"lon":37,
		 "tags":{"name":"A","station":"subway","colour":"red"}},
		{"type":"node","id":2,"lat":55,"lon":37.01,
		 "tags":{"name":"B","station":"subway","colour":"red"}},
		{"type":"node","id":3,"lat":55,"lon":37.02,
		 "tags":{"name":"C","station":"subway","colour":"red"}},
		{"type":"relation","id":90,
		 "tags":{"route":"subway","colour":"red"},
		 "members":[
			{"type":"node","ref":3,"role":"stop"},
			{"type":"node","ref":1,"role":"stop"},
			{"type":"node","ref":2,"role":"stop"}
		 ]}
	]}`

	schema, err := Parse(
		strings.NewReader(raw),
		MapWidth,
		MapHeight,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := schema.Lines[0].Stations

	if len(got) != 3 ||
		got[0] != 3 ||
		got[1] != 1 ||
		got[2] != 2 {
		t.Fatalf(
			"OSM route order = %v; want [3 1 2]",
			got,
		)
	}
}

func TestParseIncompleteDirectionsKeepFallbackStations(t *testing.T) {
	raw := `{"elements":[
		{"type":"node","id":1,"lat":55,"lon":37,
		 "tags":{"name":"A","station":"subway","colour":"red"}},
		{"type":"node","id":2,"lat":55,"lon":37.01,
		 "tags":{"name":"B","station":"subway","colour":"red"}},
		{"type":"node","id":3,"lat":55,"lon":37.02,
		 "tags":{"name":"C","station":"subway","colour":"red"}},
		{"type":"relation","id":90,
		 "tags":{"route":"subway","colour":"red","ref":"1"},
		 "members":[
			{"type":"node","ref":3,"role":"stop"},
			{"type":"node","ref":1,"role":"stop"}
		 ]},
		{"type":"relation","id":91,
		 "tags":{"route":"subway","colour":"red","ref":"1"},
		 "members":[
			{"type":"node","ref":1,"role":"stop"},
			{"type":"node","ref":2,"role":"stop"}
		 ]}
	]}`

	schema, err := Parse(
		strings.NewReader(raw),
		MapWidth,
		MapHeight,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := schema.Lines[0].Stations

	if len(got) != 3 ||
		got[0] != 1 ||
		got[1] != 2 ||
		got[2] != 3 {
		t.Fatalf(
			"incomplete route fallback order = %v; want [1 2 3]",
			got,
		)
	}
}
