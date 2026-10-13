INSERT INTO cities (name, slug)
VALUES ('Санкт-Петербург', 'saint-petersburg')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO stations (city_id, name, x, y)
SELECT c.id, s.name, s.x, s.y
FROM cities c
CROSS JOIN (VALUES
    ('Вокзальная', 90, 210),
    ('Площадь Мира', 220, 210),
    ('Невская', 350, 210),
    ('Петроградская', 480, 210),
    ('Северная', 610, 210),
    ('Морская', 350, 85),
    ('Адмиралтейская', 350, 335),
    ('Набережная', 510, 335)
) AS s(name, x, y)
WHERE c.slug = 'saint-petersburg';

INSERT INTO lines (city_id, name, color)
SELECT c.id, l.name, l.color
FROM cities c
CROSS JOIN (VALUES
    ('Красная', '#ef4444'),
    ('Синяя', '#3b82f6')
) AS l(name, color)
WHERE c.slug = 'saint-petersburg';

INSERT INTO line_stations (line_id, station_id, position)
SELECT l.id, s.id, route.position
FROM cities c
JOIN lines l ON l.city_id = c.id
JOIN (VALUES
    ('Красная', 'Вокзальная', 1),
    ('Красная', 'Площадь Мира', 2),
    ('Красная', 'Невская', 3),
    ('Красная', 'Петроградская', 4),
    ('Красная', 'Северная', 5),
    ('Синяя', 'Морская', 1),
    ('Синяя', 'Невская', 2),
    ('Синяя', 'Адмиралтейская', 3),
    ('Синяя', 'Набережная', 4)
) AS route(line_name, station_name, position)
    ON route.line_name = l.name
JOIN stations s
    ON s.city_id = c.id
    AND s.name = route.station_name
WHERE c.slug = 'saint-petersburg';