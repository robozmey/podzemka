package overpass

import (
	"os"
	"testing"
)

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
