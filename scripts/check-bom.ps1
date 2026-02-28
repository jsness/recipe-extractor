$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$files = git -C $root ls-files
if ($LASTEXITCODE -ne 0) {
  Write-Error "git ls-files failed"
  exit 1
}

$bad = @()
foreach ($f in $files) {
  $path = Join-Path $root $f
  if (-not (Test-Path $path)) { continue }
  $bytes = [System.IO.File]::ReadAllBytes($path)
  if ($bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF) {
    $bad += $f
  }
}

if ($bad.Count -gt 0) {
  Write-Host "UTF-8 BOM detected in:"
  $bad | ForEach-Object { Write-Host "  $_" }
  Write-Host "Remove BOM before committing."
  exit 1
}
