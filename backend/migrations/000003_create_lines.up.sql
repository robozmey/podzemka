CREATE TABLE lines (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    city_id BIGINT NOT NULL REFERENCES cities(id),
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    UNIQUE (city_id, name),
    UNIQUE (id, city_id)
);
