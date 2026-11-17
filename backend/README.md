### Формат JSON

Схема метро содержит два массива: `stations` и `lines`.

- `stations`: `id` (уникальный положительный ID), `name` (название), `x`, `y` (координаты), `year` (необязательный год открытия).
- `lines`: `id` (уникальный положительный ID), `name` (название), `color` (цвет `#RRGGBB`), `stations` (ID станций в порядке следования).

Все ID станций в линиях должны существовать в `stations`. Название и slug города хранятся отдельно в таблице `cities`.

### Запуск

При запущенном Docker:

```powershell
docker compose up -d db
cd backend
$env:DATABASE_URL = 'postgres://podzemka:podzemka@localhost:5433/podzemka?sslmode=disable'
$tokenBytes = New-Object byte[] 32
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($tokenBytes)
$rng.Dispose()
$env:ADMIN_TOKEN = [Convert]::ToBase64String($tokenBytes)
$env:ADMIN_TOKEN
$env:OSM_CONTACT_EMAIL = 'you@example.com'
go run ./cmd/api
```

Сгенерируй токен один раз в PowerShell и скопируй напечатанное значение в поле входа «Администрирование». Это секрет-ключ без отдельного аккаунта. Для постоянной локальной настройки скопируй `backend/.env.example` в `backend/.env` и задай собственный длинный `ADMIN_TOKEN`; файл `.env` игнорируется git. Не вставляй токен в frontend-код, README или коммиты. При запуске без `ADMIN_TOKEN` административные endpoints закрыты.

### Добавление города

Из папки `backend` при настроенном `DATABASE_URL`:

```powershell
go run ./cmd/importcity -file kazan.json -name "Казань" -slug kazan
```

Команда проверяет схему и сохраняет город в PostgreSQL. При повторном запуске с тем же slug существующая схема обновляется.

### Импорт станций из Overpass

`importstations` превращает JSON-выгрузку Overpass в формат схемы метро. Станции группируются по цвету, а порядок внутри каждой линии строится жадным алгоритмом ближайшего соседа по географическим координатам. Это приблизительное восстановление: развилки и пересадки после импорта стоит проверять вручную.

```powershell
go run ./cmd/importstations -input "C:\Users\ZonMyCar\OneDrive\Документы\metroSpb.txt" -output "$env:TEMP\saint-petersburg-stations.json"
```

Результат создаётся отдельно; команда не изменяет исходную выгрузку и не загружает JSON в базу автоматически.

### Добавление города через сайт

Нажмите «Администрирование», введите `ADMIN_TOKEN`, затем укажите только русское название города. Сервер ищет его административную границу в Nominatim, отправляет Overpass API запрос по станциям метро, показывает предпросмотр и автоматически формирует slug. После подтверждения город сохраняется в PostgreSQL и появляется в списке.

Импортированные через админку города сохраняются только в PostgreSQL (`metro_schemas.schema`, JSONB). `cmd/seedschemas/data` предназначен для версионируемых начальных схем проекта и не изменяется во время работы сайта; дублирование каждого нового города в этой папке не требуется.

Админ-доступ защищён Bearer-токеном на backend; preview и create endpoints отвечают `503`, пока `ADMIN_TOKEN` не настроен, и `401` без верного токена. Повторно занятый slug не перезаписывается: для нового города автоматически выбирается следующий свободный вариант.

Импорт использует OpenStreetMap Nominatim и Overpass API напрямую. Макрос `{{geocodeArea:...}}` предназначен для интерфейса Overpass Turbo, поэтому сервер сначала находит OSM area ID города и подставляет его в запрос Overpass. Станции группируются по цвету из OSM-тегов; порядок линии восстанавливается по ближайшим географическим координатам и может быть приблизительным. `OSM_CONTACT_EMAIL` добавляет контакт в User-Agent сервера.

Если VPN или сеть блокируют основной OSM-сервер, импорт последовательно пробует публичные зеркала. При необходимости можно задать свой совместимый endpoint в `NOMINATIM_URL` и `OVERPASS_URL` в `backend/.env`; для `OVERPASS_URL` указывается полный адрес `/api/interpreter`.

Для создания города через HTTP API отправьте `POST /api/cities` с JSON-объектом. Slug должен состоять из строчных латинских букв и цифр, разделённых одиночными дефисами.

```powershell
$body = @'
{
  "name": "Казань",
  "slug": "kazan",
  "stations": [
    { "id": 1, "name": "Кремлёвская", "x": 100, "y": 120, "year": 2005 }
  ],
  "lines": [
    { "id": 1, "name": "Центральная", "color": "#123456", "stations": [1] }
  ]
}
'@
Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/cities -ContentType 'application/json' -Body $body
```

Новый slug создаётся с ответом `201 Created`; Если город с указанным slug уже существует, API возвращает `409 Conflict` и не изменяет его схему. Некорректный JSON, slug или схема возвращают `400 Bad Request`. Схема проверяется по тем же правилам, что и при импорте JSON.
