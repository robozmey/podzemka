package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/robozmey/podzemka/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

type PostgresRepository struct {
	db *sql.DB
}

func OpenPostgres(ctx context.Context) (*sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, wrapPostgresError("ping postgres", err)
	}

	return db, nil
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListCities(ctx context.Context) ([]domain.City, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, slug
		FROM cities
		ORDER BY id
	`)
	if err != nil {
		return nil, wrapPostgresError("list cities", err)
	}
	defer rows.Close()

	var cities []domain.City
	for rows.Next() {
		var city domain.City
		if err := rows.Scan(&city.ID, &city.Name, &city.Slug); err != nil {
			return nil, wrapPostgresError("scan city", err)
		}
		cities = append(cities, city)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapPostgresError("iterate cities", err)
	}

	return cities, nil
}

func (r *PostgresRepository) GetCityBySlug(ctx context.Context, slug string) (domain.City, error) {
	var city domain.City
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, slug
		FROM cities
		WHERE slug = $1
	`, slug).Scan(&city.ID, &city.Name, &city.Slug)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.City{}, ErrNotFound
	}
	if err != nil {
		return domain.City{}, wrapPostgresError("get city by slug", err)
	}

	return city, nil
}

func (r *PostgresRepository) ListStationsByCityID(ctx context.Context, cityID int) ([]domain.Station, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, x, y, city_id
		FROM stations
		WHERE city_id = $1
		ORDER BY id
	`, cityID)
	if err != nil {
		return nil, wrapPostgresError("list stations by city id", err)
	}
	defer rows.Close()

	var stations []domain.Station
	for rows.Next() {
		var station domain.Station
		if err := rows.Scan(&station.ID, &station.Name, &station.X, &station.Y, &station.CityID); err != nil {
			return nil, wrapPostgresError("scan station", err)
		}
		stations = append(stations, station)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapPostgresError("iterate stations", err)
	}

	return stations, nil
}

func (r *PostgresRepository) ListLinesByCityID(ctx context.Context, cityID int) ([]domain.MetroLine, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.name, l.color, ls.station_id
		FROM lines l
		LEFT JOIN line_stations ls ON ls.line_id = l.id AND ls.city_id = l.city_id
		WHERE l.city_id = $1
		ORDER BY l.id, ls.position
	`, cityID)
	if err != nil {
		return nil, wrapPostgresError("list lines by city id", err)
	}
	defer rows.Close()

	var lines []domain.MetroLine
	var currentLineID int
	for rows.Next() {
		var lineID int
		var name string
		var color string
		var stationID sql.NullInt64
		if err := rows.Scan(&lineID, &name, &color, &stationID); err != nil {
			return nil, wrapPostgresError("scan line", err)
		}

		if len(lines) == 0 || currentLineID != lineID {
			lines = append(lines, domain.MetroLine{
				ID:       lineID,
				Name:     name,
				Color:    color,
				Stations: []int{},
			})
			currentLineID = lineID
		}

		if stationID.Valid {
			lines[len(lines)-1].Stations = append(lines[len(lines)-1].Stations, int(stationID.Int64))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, wrapPostgresError("iterate lines", err)
	}

	return lines, nil
}

func wrapPostgresError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("%s: postgres %s: %s", operation, pgErr.Code, pgErr.Message)
	}

	return fmt.Errorf("%s: %w", operation, err)
}
