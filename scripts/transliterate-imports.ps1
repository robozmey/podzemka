param(
  [string]$Directory = (Join-Path $PSScriptRoot '..\backend\data\imports')
)

$ErrorActionPreference = 'Stop'

$cityNames = @{
  'Beijing' = 'Пекин'; 'Shanghai' = 'Шанхай'; 'Guangzhou' = 'Гуанчжоу'; 'Shenzhen' = 'Шэньчжэнь'
  'Moscow' = 'Москва'; 'Saint Petersburg' = 'Санкт-Петербург'; 'Novosibirsk' = 'Новосибирск'
  'Nizhny Novgorod' = 'Нижний Новгород'; 'Kiev' = 'Киев'; 'Minsk' = 'Минск'
  'London' = 'Лондон'; 'Paris' = 'Париж'; 'Berlin' = 'Берлин'; 'Madrid' = 'Мадрид'
  'Rome' = 'Рим'; 'Milan' = 'Милан'; 'Tokyo' = 'Токио'; 'Osaka' = 'Осака'
  'Seoul' = 'Сеул'; 'New York City' = 'Нью-Йорк'; 'Los Angeles' = 'Лос-Анджелес'
  'Mexico City' = 'Мехико'; 'Buenos Aires' = 'Буэнос-Айрес'; 'São Paulo' = 'Сан-Паулу'
  'Hong Kong' = 'Гонконг'; 'Istanbul' = 'Стамбул'; 'Dubai' = 'Дубай'; 'Singapore' = 'Сингапур'
  'Toronto' = 'Торонто'; 'Sydney' = 'Сидней'; 'Melbourne' = 'Мельбурн'; 'Vienna' = 'Вена'
  'Prague' = 'Прага'; 'Budapest' = 'Будапешт'; 'Warsaw' = 'Варшава'; 'Amsterdam' = 'Амстердам'
  'Lisbon' = 'Лиссабон'; 'Athens' = 'Афины'; 'Stockholm' = 'Стокгольм'
  'Oslo' = 'Осло'; 'Copenhagen' = 'Копенгаген'; 'Brussels' = 'Брюссель'
}

$charMap = @{
  'a'='а'; 'b'='б'; 'c'='к'; 'd'='д'; 'e'='е'; 'f'='ф'; 'g'='г'; 'h'='х'; 'i'='и'; 'j'='дж'; 'k'='к'; 'l'='л'; 'm'='м'; 'n'='н'; 'o'='о'; 'p'='п'; 'q'='к'; 'r'='р'; 's'='с'; 't'='т'; 'u'='у'; 'v'='в'; 'w'='в'; 'x'='кс'; 'y'='й'; 'z'='з'
}

function Convert-Name([string]$value, [hashtable]$known = @{}) {
  if ([string]::IsNullOrWhiteSpace($value)) { return $value }
  $hasLatin = $value -match '[A-Za-z]'
  if (-not $hasLatin) {
    $clean = $value.Trim()
    if ($clean.Length -gt 0) { return $clean.Substring(0, 1).ToUpperInvariant() + $clean.Substring(1).ToLowerInvariant() }
    return $clean
  }
  $result = $value.Trim()
  foreach ($key in ($known.Keys | Sort-Object Length -Descending)) {
    $result = [regex]::Replace($result, "(?i)(?<![\p{L}])$([regex]::Escape($key))(?![\p{L}])", [string]$known[$key])
  }
  $result = $result -replace '(?i)\bMetro\b', 'метро'
  $result = $result -replace '(?i)\bSubway\b', 'метро'
  $result = $result -replace '(?i)\bLine\b', 'линия'
  $result = $result -replace '(?i)\bStation\b', 'станция'
  $result = $result -replace '(?i)Shch', 'Щ'
  $result = $result -replace '(?i)Sch', 'Ш'
  $result = $result -replace '(?i)Zh', 'Ж'
  $result = $result -replace '(?i)Kh', 'Х'
  $result = $result -replace '(?i)Ts', 'Ц'
  $result = $result -replace '(?i)Ch', 'Ч'
  $result = $result -replace '(?i)Sh', 'Ш'
  $result = $result -replace '(?i)Yu', 'Ю'
  $result = $result -replace '(?i)Ya', 'Я'
  $result = $result -replace '(?i)Yo', 'Ё'
  $result = $result.ToLowerInvariant()
  foreach ($char in $charMap.Keys) {
    $result = $result -replace $char, $charMap[$char]
  }
  if ($result.Length -gt 0) { return $result.Substring(0, 1).ToUpperInvariant() + $result.Substring(1) }
  return $result
}

foreach ($file in Get-ChildItem -LiteralPath $Directory -Filter '*.json') {
  $schema = Get-Content -Raw -LiteralPath $file.FullName | ConvertFrom-Json
  if (-not $schema.city) {
    $schema | Add-Member -MemberType NoteProperty -Name city -Value ([pscustomobject]@{ id = 0; name = [io.path]::GetFileNameWithoutExtension($file.Name); slug = '' })
  }
  $originalCity = [string]$schema.city.name
  if ($cityNames.ContainsKey($originalCity)) { $schema.city.name = $cityNames[$originalCity] }
  elseif ($originalCity -notmatch '[А-Яа-яЁё]') { $schema.city.name = Convert-Name $originalCity $cityNames }
  $schema.city.slug = ([string]$schema.city.name).ToLowerInvariant() -replace '[^\p{L}\p{Nd}]+', '-'
  foreach ($station in @($schema.stations)) { $station.name = Convert-Name ([string]$station.name) $cityNames }
  foreach ($line in @($schema.lines)) { $line.name = Convert-Name ([string]$line.name) $cityNames }
  $json = $schema | ConvertTo-Json -Depth 20
  [IO.File]::WriteAllText($file.FullName, $json, [Text.UTF8Encoding]::new($false))
}

Write-Host "Transliterated $((Get-ChildItem -LiteralPath $Directory -Filter '*.json').Count) schemas"
