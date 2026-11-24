package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/robozmey/podzemka/backend/internal/overpass"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: fetchcity <city> <output.json>")
	}

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	city := os.Args[1]
	output := os.Args[2]

	schema, err := overpass.FetchCity(context.Background(), city)
	if err != nil {
		log.Fatal(err)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"Imported %s: %d stations, %d lines",
		city,
		len(schema.Stations),
		len(schema.Lines),
	)
}
