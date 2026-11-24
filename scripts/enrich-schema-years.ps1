param(
  [string]$Directory = (Join-Path $PSScriptRoot '..\backend\data\imports'),
  [string]$File,
  [string]$OverpassUrl = 'https://overpass-api.de/api/interpreter',
  [double]$MatchTolerance = 0.004,
  [int]$DelaySeconds = 3
)

$ErrorActionPreference = 'Stop'
$utf8NoBom = [Text.UTF8Encoding]::new($false)

function Get-Year([object]$value) {
  if ($null -eq $value) { return $null }
  $match = [regex]::Match([string]$value, '(?<!\d)(18|19|20)\d{2}(?!\d)')
  if (-not $match.Success) { return $null }
  $year = [int]$match.Value
  if ($year -lt 1863 -or $year -gt (Get-Date).Year + 10) { return $null }
  return $year
}

function Get-StationYearData([object]$schema) {
  $stations = @($schema.stations | Where-Object { $null -ne $_.lat -and $null -ne $_.lon })
  if ($stations.Count -eq 0) { return @{} }
  $minLat = ($stations | Measure-Object lat -Minimum).Minimum - 0.02
  $maxLat = ($stations | Measure-Object lat -Maximum).Maximum + 0.02
  $minLon = ($stations | Measure-Object lon -Minimum).Minimum - 0.02
  $maxLon = ($stations | Measure-Object lon -Maximum).Maximum + 0.02
  $query = @"
[out:json][timeout:60];
(
  node["railway"="station"]["station"="subway"]($minLat,$minLon,$maxLat,$maxLon);
  node["railway"="station"]["subway"="yes"]($minLat,$minLon,$maxLat,$maxLon);
);
out body;
"@
  $endpoints = @($OverpassUrl, 'https://overpass.kumi.systems/api/interpreter', 'https://overpass.private.coffee/api/interpreter') | Select-Object -Unique
  $response = $null
  foreach ($endpoint in $endpoints) {
    for ($attempt = 1; $attempt -le 3; $attempt++) {
      try {
        $response = Invoke-RestMethod -Method Post -Uri $endpoint -ContentType 'application/x-www-form-urlencoded' -Body @{ data = $query } -Headers @{ 'User-Agent' = 'Podzemka station year importer' } -TimeoutSec 75
        break
      } catch {
        if ($attempt -lt 3) { Start-Sleep -Seconds ([math]::Min(20, $attempt * 5)) }
      }
    }
    if ($null -ne $response) { break }
  }
  if ($null -eq $response) { throw 'все Overpass endpoints вернули ошибку' }
  $result = @{}
  foreach ($element in @($response.elements)) {
    $year = Get-Year $element.tags.start_date
    if ($null -ne $year -and $null -ne $element.lat -and $null -ne $element.lon) {
      $result[$element.id] = [pscustomobject]@{ lat = [double]$element.lat; lon = [double]$element.lon; year = $year }
    }
  }
  return $result
}

function Find-NearestYear([object]$station, [hashtable]$osmStations) {
  $best = $null; $bestDistance = [double]::PositiveInfinity
  foreach ($candidate in $osmStations.Values) {
    $dLat = [double]$station.lat - $candidate.lat
    $dLon = [double]$station.lon - $candidate.lon
    $distance = [math]::Sqrt($dLat * $dLat + $dLon * $dLon)
    if ($distance -lt $bestDistance) { $best = $candidate; $bestDistance = $distance }
  }
  if ($null -ne $best -and $bestDistance -le $MatchTolerance) { return $best }
  return $null
}

$files = if ($File) { @(Get-Item -LiteralPath $File) } else { @(Get-ChildItem -LiteralPath $Directory -Filter '*.json' -File) }
$updatedFiles = 0; $updatedStations = 0
foreach ($inputFile in $files) {
  $schema = Get-Content -Raw -LiteralPath $inputFile.FullName | ConvertFrom-Json
  $missing = @($schema.stations | Where-Object { $null -eq $_.year -and $null -ne $_.lat -and $null -ne $_.lon })
  if ($missing.Count -eq 0) { Write-Host "$($inputFile.Name): нет станций для обогащения"; continue }
  try { $osmStations = Get-StationYearData $schema } catch { Write-Warning "$($inputFile.Name): Overpass недоступен: $($_.Exception.Message)"; continue }
  $fileUpdated = 0
  foreach ($station in $missing) {
    $match = Find-NearestYear $station $osmStations
    if ($null -ne $match) { $station | Add-Member NoteProperty year $match.year -Force; $fileUpdated++ }
  }
  if ($fileUpdated -gt 0) {
    [IO.File]::WriteAllText($inputFile.FullName, ($schema | ConvertTo-Json -Depth 20), $utf8NoBom)
    $updatedFiles++; $updatedStations += $fileUpdated
  }
  Write-Host "$($inputFile.Name): найдено годов $fileUpdated из $($missing.Count)"
  if ($DelaySeconds -gt 0) { Start-Sleep -Seconds $DelaySeconds }
}
Write-Host "Обновлено файлов: $updatedFiles; станций с годами: $updatedStations"
