param(
  [string]$OutputDirectory = (Join-Path $PSScriptRoot '..\backend\data\imports'),
  [int]$Limit = 100,
  [switch]$Force,
  [switch]$OnlyForeign,
  [string]$City
)

$ErrorActionPreference = 'Stop'
$endpoint = 'https://services7.arcgis.com/35Z9xvnYF2hyWrzH/ArcGIS/rest/services/WRN_2020/FeatureServer/0/query'
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

$fields = 'Station_Name_Eng,StationNameNative,Transit_System,Metro_Area,Line,Rail_Type,Status,Country,City'
$records = @()
$offset = 0
do {
  $query = @{
    where = "Status NOT LIKE '%construction%' AND Status NOT LIKE '%proposed%'"
    outFields = $fields
    outSR = '4326'
    returnGeometry = 'true'
    resultOffset = $offset
    resultRecordCount = 2000
    f = 'json'
  }
  $page = Invoke-RestMethod -Uri ($endpoint + '?' + (($query.GetEnumerator() | ForEach-Object { '{0}={1}' -f $_.Key, [uri]::EscapeDataString([string]$_.Value) }) -join '&'))
  if ($page.error) { throw ($page.error.message) }
  $records += @($page.features)
  $offset += @($page.features).Count
  Write-Host "Downloaded $offset station records"
} while ($page.exceededTransferLimit -or @($page.features).Count -eq 2000)

$validRecords = $records | Where-Object {
  $_.attributes.Metro_Area -and $_.attributes.Line -and $_.geometry.x -ne $null -and $_.geometry.y -ne $null
}

$cities = $validRecords | Group-Object { $_.attributes.Metro_Area } | Where-Object {
  $usableLines = @($_.Group | Group-Object { $_.attributes.Line } | Where-Object {
    @($_.Group | ForEach-Object { $_.attributes.Station_Name_Eng } | Sort-Object -Unique).Count -ge 2
  })
  $usableLines.Count -ge 2
} | Sort-Object Name | Select-Object -First $Limit

if ($City) {
  $cities = @($cities | Where-Object { $_.Name -eq $City })
  if ($cities.Count -eq 0) { throw "Город «$City» не найден в каталоге WRN" }
}

Write-Host "Valid records: $(@($validRecords).Count); candidate cities: $(@($cities).Count)"

if ($cities.Count -lt $Limit) {
  Write-Warning "Источник предоставил только $($cities.Count) городов с двумя и более линиями"
}

function Normalize-Name([string]$value) {
  $safe = ($value -replace '[^\p{L}\p{Nd}]+', '-').Trim('-')
  if ([string]::IsNullOrWhiteSpace($safe)) { $safe = 'city' }
  return $safe
}

$charMap = @{
  'a'='а'; 'b'='б'; 'c'='к'; 'd'='д'; 'e'='е'; 'f'='ф'; 'g'='г'; 'h'='х'; 'i'='и'; 'j'='дж'; 'k'='к'; 'l'='л'; 'm'='м'; 'n'='н'; 'o'='о'; 'p'='п'; 'q'='к'; 'r'='р'; 's'='с'; 't'='т'; 'u'='у'; 'v'='в'; 'w'='в'; 'x'='кс'; 'y'='й'; 'z'='з'
}

function Transliterate([string]$value) {
  $result = $value.Trim()
  $result = $result.Replace('ł', 'l').Replace('Ł', 'L').Replace('ą', 'a').Replace('Ą', 'A').Replace('ę', 'e').Replace('Ę', 'E').Replace('ó', 'u').Replace('Ó', 'U').Replace('ś', 's').Replace('Ś', 'S').Replace('ź', 'z').Replace('Ź', 'Z').Replace('ż', 'z').Replace('Ż', 'Z').Replace('ć', 'c').Replace('Ć', 'C').Replace('ń', 'n').Replace('Ń', 'N')
  $result = $result -replace '(?i)\bMetro\b', 'метро'
  $result = $result -replace '(?i)\bSubway\b', 'метро'
  $result = $result -replace '(?i)\bLine\b', 'линия'
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
  foreach ($char in $charMap.Keys) { $result = $result -replace $char, $charMap[$char] }
  if ($result.Length -eq 0) { return $result }
  return $result.Substring(0, 1).ToUpperInvariant() + $result.Substring(1)
}

function Foreign-Name([string]$value) {
  $original = $value.Trim()
  if ([string]::IsNullOrWhiteSpace($original)) { return $original }
  return "$original ($(Transliterate $original))"
}

$russianCities = @{
  'Moscow' = 'Москва'
  'Saint Petersburg' = 'Санкт-Петербург'
  'Novosibirsk' = 'Новосибирск'
  'Nizhny Novgorod' = 'Нижний Новгород'
  'Kiev' = 'Киев'
  'Minsk' = 'Минск'
}

function Distance($a, $b) {
  return [math]::Sqrt([math]::Pow($a.geometry.x - $b.geometry.x, 2) + [math]::Pow($a.geometry.y - $b.geometry.y, 2))
}

function Order-Stations($items) {
  $remaining = [System.Collections.Generic.List[object]]::new()
  foreach ($item in $items) { $remaining.Add($item) }
  $ordered = [System.Collections.Generic.List[object]]::new()
  $current = $remaining | Sort-Object { $_.geometry.x }, { $_.geometry.y } | Select-Object -First 1
  while ($current) {
    $ordered.Add($current)
    [void]$remaining.Remove($current)
    if ($remaining.Count -eq 0) { break }
    $next = $remaining | Sort-Object { Distance $_ $current } | Select-Object -First 1
    $current = $next
  }
  return $ordered
}

$cityNumber = 0
foreach ($city in $cities) {
  $metroArea = [string]$city.Name
  if ($OnlyForeign -and $russianCities.ContainsKey($metroArea)) { continue }
  $displayName = if ($russianCities.ContainsKey($metroArea)) { $russianCities[$metroArea] } else { $metroArea }
  if (-not $russianCities.ContainsKey($metroArea)) { $displayName = Foreign-Name $metroArea }
  $cityRecords = @($records | Where-Object { [string]$_.attributes.Metro_Area -eq $metroArea })
  $stationMap = @{}
  $stations = [System.Collections.Generic.List[object]]::new()
  $lines = [System.Collections.Generic.List[object]]::new()
  $lineNumber = 0
  foreach ($lineGroup in ($cityRecords | Where-Object { $_.attributes.Line } | Group-Object { $_.attributes.Line } | Sort-Object Name)) {
    $lineStations = @($lineGroup.Group | Sort-Object { $_.attributes.Station_Name_Eng })
    if ($lineStations.Count -lt 2) { continue }
    $ordered = Order-Stations $lineStations
    $stationIds = [System.Collections.Generic.List[int]]::new()
    foreach ($item in $ordered) {
      $key = '{0}|{1}|{2}' -f $item.attributes.Station_Name_Eng, $item.geometry.x, $item.geometry.y
      if (-not $stationMap.ContainsKey($key)) {
        $id = $stations.Count + 1
        $stationMap[$key] = $id
        $lat = [double]$item.geometry.y
        $lon = [double]$item.geometry.x
        $originalStationName = if ($item.attributes.Station_Name_Eng) { [string]$item.attributes.Station_Name_Eng } else { [string]$item.attributes.StationNameNative }
        $stationName = if ($russianCities.ContainsKey($metroArea) -and $item.attributes.StationNameNative) { $item.attributes.StationNameNative } elseif ($russianCities.ContainsKey($metroArea)) { $originalStationName } else { Foreign-Name $originalStationName }
        $stations.Add([pscustomobject][ordered]@{ id = $id; name = $stationName; x = 0; y = 0; lat = $lat; lon = $lon })
      }
      $stationIds.Add($stationMap[$key])
    }
    $lineName = if ($russianCities.ContainsKey($metroArea)) { [string]$lineGroup.Name } else { Foreign-Name ([string]$lineGroup.Name) }
    $lines.Add([pscustomobject][ordered]@{ id = $lines.Count + 1; name = $lineName; color = ('#{0:X6}' -f ((($lines.Count + 1) * 2654435761) -band 0xFFFFFF)); stations = @($stationIds.ToArray()) })
    $lineNumber++
  }
  if ($stations.Count -lt 4 -or $lines.Count -lt 2) {
    continue
  }
  $stationArray = $stations.ToArray()
  $minX = ($stationArray | Measure-Object lon -Minimum).Minimum; $maxX = ($stationArray | Measure-Object lon -Maximum).Maximum
  $minY = ($stationArray | Measure-Object lat -Minimum).Minimum; $maxY = ($stationArray | Measure-Object lat -Maximum).Maximum
  foreach ($station in $stationArray) {
    $station.x = [math]::Round(60 + (($station.lon - $minX) / [math]::Max(0.0001, $maxX - $minX)) * 880)
    $station.y = [math]::Round(60 + (1 - (($station.lat - $minY) / [math]::Max(0.0001, $maxY - $minY))) * 580)
  }
  $cityNumber++
  $file = Join-Path $OutputDirectory ((Normalize-Name $metroArea) + '.json')
  if ((Test-Path -LiteralPath $file) -and -not $Force) { continue }
  [ordered]@{ city = [ordered]@{ id = $cityNumber; name = $displayName; slug = (Normalize-Name $metroArea).ToLowerInvariant() }; stations = @($stationArray); lines = @($lines.ToArray()) } | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $file -Encoding utf8
  Write-Host "SAVED ${metroArea}: stations=$($stations.Count), lines=$($lines.Count)" -ForegroundColor Green
}

Write-Host "Created $cityNumber schemas in $OutputDirectory"
