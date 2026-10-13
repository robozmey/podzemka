ALTER TABLE stations
    ADD CONSTRAINT stations_id_city_id_key UNIQUE (id, city_id);

CREATE TABLE line_stations (
    line_id BIGINT NOT NULL,
    station_id BIGINT NOT NULL,
    city_id BIGINT NOT NULL,
    position INTEGER NOT NULL CHECK (position > 0),
    PRIMARY KEY (line_id, station_id),
    UNIQUE (line_id, position),
    FOREIGN KEY (line_id, city_id) REFERENCES lines(id, city_id) ON DELETE CASCADE,
    FOREIGN KEY (station_id, city_id) REFERENCES stations(id, city_id)
);
