INSERT INTO cities (name, slug)
VALUES
    ('Санкт-Петербург', 'saint-petersburg'),
    ('Москва', 'moscow'),
    ('Красноярск', 'krasnoyarsk')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO stations (city_id, name, x, y)
SELECT c.id, s.name, s.x, s.y
FROM (VALUES
    ('saint-petersburg', 'Вокзальная', 90, 210),
    ('saint-petersburg', 'Площадь Мира', 220, 210),
    ('saint-petersburg', 'Невская', 350, 210),
    ('saint-petersburg', 'Петроградская', 480, 210),
    ('saint-petersburg', 'Северная', 610, 210),
    ('saint-petersburg', 'Морская', 350, 85),
    ('saint-petersburg', 'Адмиралтейская', 350, 335),
    ('saint-petersburg', 'Набережная', 510, 335),

    ('moscow', 'Белорусская', 90, 210),
    ('moscow', 'Маяковская', 220, 210),
    ('moscow', 'Тверская', 350, 210),
    ('moscow', 'Театральная', 480, 210),
    ('moscow', 'Курская', 610, 210),
    ('moscow', 'Краснопресненская', 350, 85),
    ('moscow', 'Парк культуры', 350, 335),
    ('moscow', 'Киевская', 220, 85),

    ('krasnoyarsk', 'Вокзальная', 90, 210),
    ('krasnoyarsk', 'Площадь Революции', 220, 210),
    ('krasnoyarsk', 'Театральная', 350, 210),
    ('krasnoyarsk', 'Енисейская', 480, 210),
    ('krasnoyarsk', 'Правобережная', 610, 210),
    ('krasnoyarsk', 'Университет', 350, 85),
    ('krasnoyarsk', 'Солнечная', 350, 335),
    ('krasnoyarsk', 'Зелёная Роща', 480, 335)
) AS s(city_slug, name, x, y)
JOIN cities c ON c.slug = s.city_slug;

INSERT INTO lines (city_id, name, color)
SELECT c.id, l.name, l.color
FROM (VALUES
    ('saint-petersburg', 'Красная', '#ef4444'),
    ('saint-petersburg', 'Синяя', '#3b82f6'),
    ('moscow', 'Центральная', '#ef4444'),
    ('moscow', 'Радиальная', '#3b82f6'),
    ('krasnoyarsk', 'Центральная', '#22c55e'),
    ('krasnoyarsk', 'Енисейская', '#f97316')
) AS l(city_slug, name, color)
JOIN cities c ON c.slug = l.city_slug;

INSERT INTO line_stations (line_id, station_id, position)
SELECT l.id, s.id, route.position
FROM (VALUES
    ('saint-petersburg', 'Красная', 'Вокзальная', 1),
    ('saint-petersburg', 'Красная', 'Площадь Мира', 2),
    ('saint-petersburg', 'Красная', 'Невская', 3),
    ('saint-petersburg', 'Красная', 'Петроградская', 4),
    ('saint-petersburg', 'Красная', 'Северная', 5),
    ('saint-petersburg', 'Синяя', 'Морская', 1),
    ('saint-petersburg', 'Синяя', 'Невская', 2),
    ('saint-petersburg', 'Синяя', 'Адмиралтейская', 3),
    ('saint-petersburg', 'Синяя', 'Набережная', 4),

    ('moscow', 'Центральная', 'Белорусская', 1),
    ('moscow', 'Центральная', 'Маяковская', 2),
    ('moscow', 'Центральная', 'Тверская', 3),
    ('moscow', 'Центральная', 'Театральная', 4),
    ('moscow', 'Центральная', 'Курская', 5),
    ('moscow', 'Радиальная', 'Киевская', 1),
    ('moscow', 'Радиальная', 'Краснопресненская', 2),
    ('moscow', 'Радиальная', 'Тверская', 3),
    ('moscow', 'Радиальная', 'Парк культуры', 4),

    ('krasnoyarsk', 'Центральная', 'Вокзальная', 1),
    ('krasnoyarsk', 'Центральная', 'Площадь Революции', 2),
    ('krasnoyarsk', 'Центральная', 'Театральная', 3),
    ('krasnoyarsk', 'Центральная', 'Енисейская', 4),
    ('krasnoyarsk', 'Центральная', 'Правобережная', 5),
    ('krasnoyarsk', 'Енисейская', 'Университет', 1),
    ('krasnoyarsk', 'Енисейская', 'Театральная', 2),
    ('krasnoyarsk', 'Енисейская', 'Солнечная', 3),
    ('krasnoyarsk', 'Енисейская', 'Зелёная Роща', 4)
) AS route(city_slug, line_name, station_name, position)
JOIN cities c ON c.slug = route.city_slug
JOIN lines l ON l.city_id = c.id AND l.name = route.line_name
JOIN stations s ON s.city_id = c.id AND s.name = route.station_name;