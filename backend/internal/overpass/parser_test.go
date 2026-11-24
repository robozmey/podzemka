package overpass

import (
	"os"
	"strings"
	"testing"
)

func TestParseUsesSubwayRouteWays(t *testing.T) {
	input := `{"elements":[
{"type":"node","id":1,"lat":55.70,"lon":37.60,"tags":{"railway":"station","station":"subway","name:en":"North"}},
{"type":"node","id":2,"lat":55.71,"lon":37.61,"tags":{"railway":"station","station":"subway","name:en":"South"}},
{"type":"way","id":100,"nodes":[1,2]},
{"type":"relation","id":200,"tags":{"type":"route","route":"subway","ref":"1","name:en":"Red Line","colour":"red"},"members":[{"type":"way","ref":100,"role":""}]}
]}`
	schema, err := Parse(strings.NewReader(input), MapWidth, MapHeight)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Lines) != 1 || len(schema.Lines[0].Stations) != 2 {
		t.Fatalf("lines=%d stations on line=%d, want 1 and 2", len(schema.Lines), len(schema.Lines[0].Stations))
	}
}

func TestParseKazanLooseOverpassExport(t *testing.T) {
	file, err := os.Open(`C:\Users\ZonMyCar\Заначка\Текстовый документ.json`)
	if err != nil {
		t.Skipf("user Overpass export unavailable: %v", err)
	}
	defer file.Close()

	schema, err := Parse(file, MapWidth, MapHeight)
	if err != nil {
		t.Fatalf("Parse Kazan export: %v", err)
	}
	if len(schema.Stations) != 11 {
		t.Fatalf("stations=%d, want 11", len(schema.Stations))
	}
}
