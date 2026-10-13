BEGIN;

INSERT INTO cities (id, name, slug)
OVERRIDING SYSTEM VALUE
VALUES
    (1, 'Санкт-Петербург', 'saint-petersburg');

INSERT INTO stations (id, name, x, y, city_id)
OVERRIDING SYSTEM VALUE
VALUES
    (1, 'Вокзальная', 90, 210, 1),
    (2, 'Площадь Мира', 220, 210, 1),
    (3, 'Невская', 350, 210, 1),
    (4, 'Петроградская', 480, 210, 1),
    (5, 'Северная', 610, 210, 1),
    (6, 'Морская', 350, 85, 1),
    (7, 'Адмиралтейская', 350, 335, 1),
    (8, 'Набережная', 510, 335, 1);

INSERT INTO lines (id, city_id, name, color)
OVERRIDING SYSTEM VALUE
VALUES
    (1, 1, 'Красная', '#ef4444'),
    (2, 1, 'Синяя', '#3b82f6');

INSERT INTO line_stations (line_id, station_id, city_id, position)
VALUES
    (1, 1, 1, 1),
    (1, 2, 1, 2),
    (1, 3, 1, 3),
    (1, 4, 1, 4),
    (1, 5, 1, 5),
    (2, 6, 1, 1),
    (2, 3, 1, 2),
    (2, 7, 1, 3),
    (2, 8, 1, 4);

SELECT setval(pg_get_serial_sequence('cities', 'id'), COALESCE((SELECT MAX(id) FROM cities), 1), (SELECT COUNT(*) > 0 FROM cities));
SELECT setval(pg_get_serial_sequence('stations', 'id'), COALESCE((SELECT MAX(id) FROM stations), 1), (SELECT COUNT(*) > 0 FROM stations));
SELECT setval(pg_get_serial_sequence('lines', 'id'), COALESCE((SELECT MAX(id) FROM lines), 1), (SELECT COUNT(*) > 0 FROM lines));

COMMIT;
