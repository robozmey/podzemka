$ErrorActionPreference = 'Stop'
$jsonPath = Join-Path $PSScriptRoot '..\backend\data\imports\Moscow.json'
$sourceUrl = 'https://metro2.org/msk/stations'

function Normalize([string]$value) {
  if ([string]::IsNullOrWhiteSpace($value)) { return '' }
  return (($value.ToLowerInvariant() -replace 'ё', 'е') -replace '[^а-яa-z0-9]+', '')
}

$html = (Invoke-WebRequest -Uri $sourceUrl -UseBasicParsing).Content
$rows = [regex]::Matches($html, '<tr>(?<row>.*?)</tr>', [Text.RegularExpressions.RegexOptions]::Singleline)
$dateByStation = @{}
$dateByLineStation = @{}

foreach ($match in $rows) {
  $row = $match.Groups['row'].Value
  $nameMatch = [regex]::Match($row, 'title-with-label-inline.*?data-label="[^"]*"></span>\s*<span>(?<name>[^<]+)</span>', [Text.RegularExpressions.RegexOptions]::Singleline)
  $lineMatch = [regex]::Match($row, 'views-field-field-line[^>]*>(?<line>.*?)</td>', [Text.RegularExpressions.RegexOptions]::Singleline)
  $dateMatch = [regex]::Match($row, 'views-field-date-open-with-fallback[^>]*>(?<date>.*?)</td>', [Text.RegularExpressions.RegexOptions]::Singleline)
  if (-not ($nameMatch.Success -and $lineMatch.Success -and $dateMatch.Success)) { continue }

  $name = [Net.WebUtility]::HtmlDecode($nameMatch.Groups['name'].Value).Trim()
  $line = ([Net.WebUtility]::HtmlDecode($lineMatch.Groups['line'].Value) -replace '<[^>]+>', '').Trim()
  $yearMatch = [regex]::Match($dateMatch.Groups['date'].Value, '\b(18|19|20)\d{2}\b')
  if (-not $yearMatch.Success) { continue }
  $year = [int]$yearMatch.Value
  $stationKey = Normalize $name
  $lineKey = Normalize $line
  if (-not $dateByStation.ContainsKey($stationKey) -or $year -lt $dateByStation[$stationKey]) { $dateByStation[$stationKey] = $year }
  $dateByLineStation["$lineKey|$stationKey"] = $year
}

$schema = Get-Content -Raw -LiteralPath $jsonPath | ConvertFrom-Json
$matched = 0
foreach ($line in $schema.lines) {
  $lineName = [string]$line.name -replace '\s+линия$', ''
  if ($lineName -eq 'Калининская' -or $lineName -eq 'Солнцевская') { $lineName = 'Калининско-Солнцевская' }
  $lineKey = Normalize $lineName
  foreach ($stationId in @($line.stations)) {
    $station = $schema.stations | Where-Object id -eq $stationId | Select-Object -First 1
    if (-not $station) { continue }
    $stationKey = Normalize ([string]$station.name)
    $key = "$lineKey|$stationKey"
    $year = if ($dateByLineStation.ContainsKey($key)) { $dateByLineStation[$key] } elseif ($dateByStation.ContainsKey($stationKey)) { $dateByStation[$stationKey] } else { $null }
    if ($null -ne $year) {
      $station | Add-Member -MemberType NoteProperty -Name year -Value $year -Force
      $matched++
    }
  }
}

$json = $schema | ConvertTo-Json -Depth 20
[IO.File]::WriteAllText((Resolve-Path $jsonPath), $json, [Text.UTF8Encoding]::new($false))
Write-Host "Добавлены годы для $matched связей станций; источник: $sourceUrl"
