param(
  [string]$ApiUrl = "http://localhost:8080",
  [string]$AdminToken = $env:ADMIN_TOKEN,
  [int]$Retries = 3,
  [int]$DelaySeconds = 3,
  [string[]]$CityNames,
  [switch]$Force
)

if ([string]::IsNullOrWhiteSpace($AdminToken)) {
  throw "Передайте токен: `$env:ADMIN_TOKEN = 'dev-token'"
}

$outputDirectory = Join-Path $PSScriptRoot "..\backend\data\imports"
New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null

$headers = @{ Authorization = "Bearer $AdminToken" }
$cities = Invoke-RestMethod -Uri "$ApiUrl/api/catalog/cities" -Headers $headers
if ($CityNames -and $CityNames.Count -gt 0) {
  $wanted = @{}
  foreach ($cityName in (($CityNames -join ',') -split ',')) {
    $wanted[$cityName.Trim().ToLowerInvariant()] = $true
  }
  $cities = @($cities | Where-Object {
    $wanted.ContainsKey(([string]$_.name).Trim().ToLowerInvariant())
  })
  if ($cities.Count -eq 0) {
    throw "Не найдено городов для выгрузки: $($CityNames -join ', ')"
  }
}
$existing = Invoke-RestMethod -Uri "$ApiUrl/api/cities" -Headers $headers
$existingNames = @{}
foreach ($city in $existing) {
  $existingNames[$city.name.Trim().ToLowerInvariant()] = $true
}

foreach ($city in $cities) {
  $name = [string]$city.name
  $searchName = if ([string]::IsNullOrWhiteSpace([string]$city.query)) { $name } else { [string]$city.query }
  $safeName = ($name -replace '[^\p{L}\p{Nd}]+', '-').Trim('-')
  $filePath = Join-Path $outputDirectory "$safeName.json"

  if (-not $Force -and (Test-Path -LiteralPath $filePath)) {
    try {
      $stored = Get-Content -Raw -LiteralPath $filePath | ConvertFrom-Json
      $storedStations = @($stored.stations).Count
      $storedLines = @($stored.lines).Count
      if ($storedStations -gt 0 -and $storedLines -gt 0 -and -not ($storedStations -ge 20 -and $storedLines -lt 2)) {
        Write-Host "SKIP usable file: $name ($storedStations stations, $storedLines lines)"
        continue
      }
      Write-Warning "RETRY invalid existing file: $name ($storedStations stations, $storedLines lines)"
    } catch {
      Write-Warning "RETRY unreadable existing file: $name"
    }
  }

  if (-not $Force -and $existingNames.ContainsKey($name.Trim().ToLowerInvariant())) {
    Write-Host "SKIP existing: $name"
    continue
  }

  $body = [Text.Encoding]::UTF8.GetBytes((ConvertTo-Json @{ name = $searchName } -Compress))
  $saved = $false
  for ($attempt = 1; $attempt -le $Retries -and -not $saved; $attempt++) {
    try {
      Write-Host "[$attempt/$Retries] Loading: $name"
      $preview = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiUrl/api/admin/cities/preview" `
        -Headers $headers `
        -ContentType 'application/json; charset=utf-8' `
        -Body $body `
        -TimeoutSec 180

      $stationCount = @($preview.schema.stations).Count
      $lineCount = @($preview.schema.lines).Count
      if ($stationCount -eq 0 -or $lineCount -eq 0 -or ($stationCount -ge 20 -and $lineCount -lt 2)) {
        throw "получена подозрительная схема: $stationCount станций, $lineCount линий"
      }
      $preview.schema | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $filePath -Encoding utf8
      Write-Host "SAVED: $filePath ($stationCount stations, $lineCount lines)" -ForegroundColor Green
      $saved = $true
    } catch {
      Write-Warning "${name}: $($_.Exception.Message)"
      if ($attempt -lt $Retries) {
        Start-Sleep -Seconds $DelaySeconds
      }
    }
  }
}
