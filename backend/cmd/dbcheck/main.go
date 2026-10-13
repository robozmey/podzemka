package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robozmey/podzemka/backend/internal/repository"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := repository.OpenPostgres(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewPostgresRepository(db)

	cities, err := repo.ListCities(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("cities: %d\n", len(cities))
	for _, city := range cities {
		stations, err := repo.ListStationsByCityID(ctx, city.ID)
		if err != nil {
			log.Fatal(err)
		}

		lines, err := repo.ListLinesByCityID(ctx, city.ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%d %s (%s): %d stations, %d lines\n", city.ID, city.Name, city.Slug, len(stations), len(lines))
		for _, line := range lines {
			fmt.Printf("  line %d %s %s: %v\n", line.ID, line.Name, line.Color, line.Stations)
		}
	}
}
