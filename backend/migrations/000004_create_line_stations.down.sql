DROP TABLE IF EXISTS line_stations;

ALTER TABLE stations
    DROP CONSTRAINT IF EXISTS stations_id_city_id_key;
