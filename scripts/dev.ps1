$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $root '.env'
$devComposeFile = Join-Path $root 'compose.dev.yml'
$devComposeProject = if ($env:DEV_COMPOSE_PROJECT) { $env:DEV_COMPOSE_PROJECT } else { 'recipe-extractor-local' }

function Resolve-DevGo {
  $devGoCommand = Get-Command go -CommandType Application -ErrorAction SilentlyContinue
  if ($devGoCommand) { return $devGoCommand.Source }
  foreach ($devGoCandidate in @(
    (Join-Path $env:ProgramFiles 'Go\bin\go.exe'),
    (Join-Path $env:LOCALAPPDATA 'Programs\Go\bin\go.exe')
  )) {
    if (Test-Path -LiteralPath $devGoCandidate -PathType Leaf) { return $devGoCandidate }
  }
  throw 'Go was not found. Install Go 1.24+ or add its bin directory to PATH.'
}

function Set-EnvFromFile {
  param(
    [string]$Path
  )

  if (-not (Test-Path $Path)) {
    return
  }

  Get-Content $Path | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') {
      return
    }

    $parts = $_ -split '=', 2
    if ($parts.Length -ne 2) {
      return
    }

    $name = $parts[0].Trim()
    $value = $parts[1].Trim()

    Set-Item -Path "Env:$name" -Value $value
  }
}

function Wait-ForPostgres {
  param(
    [int]$TimeoutSeconds = 60
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)

  while ((Get-Date) -lt $deadline) {
    $status = docker compose -p $devComposeProject -f $devComposeFile ps postgres 2>$null
    if ($LASTEXITCODE -eq 0 -and ($status -match 'healthy')) {
      return
    }

    Start-Sleep -Seconds 2
  }

  throw 'Postgres did not become healthy in time.'
}

Set-EnvFromFile -Path $envPath

$devHttpAddr = if ($env:DEV_HTTP_ADDR) { $env:DEV_HTTP_ADDR } else { ':8081' }
$devVitePort = if ($env:DEV_VITE_PORT) { $env:DEV_VITE_PORT } else { '5174' }
$devPostgresPort = if ($env:DEV_POSTGRES_PORT) { $env:DEV_POSTGRES_PORT } else { '5434' }

if ($devHttpAddr -notmatch ':(\d+)$') {
  throw 'DEV_HTTP_ADDR must end with a port, for example :8081.'
}
$devHttpPort = [int]$Matches[1]
if ($devHttpPort -eq 8080) {
  throw 'Port 8080 is reserved for the deployed app. Use a separate DEV_HTTP_ADDR, such as :8081.'
}

# Keep each checkout's binaries and logs separate, outside the repository.
$devPathHasher = [System.Security.Cryptography.SHA256]::Create()
try {
  $devPathHash = [BitConverter]::ToString($devPathHasher.ComputeHash(
    [Text.Encoding]::UTF8.GetBytes($root.ToLowerInvariant())
  )).Replace('-', '').Substring(0, 12)
} finally {
  $devPathHasher.Dispose()
}
$devRuntimeDir = Join-Path $env:TEMP "recipe-extractor-dev-$devPathHash"
New-Item -ItemType Directory -Path $devRuntimeDir -Force | Out-Null
$devServerBinary = Join-Path $devRuntimeDir ("server-" + [guid]::NewGuid().ToString('N') + '.exe')

Write-Host 'Building updated Go server...'
$devGoExecutable = Resolve-DevGo
Push-Location (Join-Path $root 'server')
try {
  & $devGoExecutable build -o $devServerBinary ./cmd/server
  if ($LASTEXITCODE -ne 0) { throw 'Go build failed; the existing dev server has been left running.' }
} finally {
  Pop-Location
}

$devServerOwners = @(Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
  Where-Object { $_.LocalPort -eq $devHttpPort } |
  Select-Object -ExpandProperty OwningProcess -Unique)
foreach ($devServerOwner in $devServerOwners) {
  $devServerProcess = Get-CimInstance Win32_Process -Filter "ProcessId = $devServerOwner"
  $devExecutable = $devServerProcess.ExecutablePath
  $devManagedBinary = $devExecutable -and
    (Split-Path -Parent $devExecutable) -eq $devRuntimeDir
  $devLegacyBinary = $devExecutable -and (
    $devExecutable -eq (Join-Path $env:TEMP 'recipe-extractor-notices-dev.exe') -or
    $devExecutable -eq (Join-Path $env:TEMP 'recipe-extractor-reminders-dev.exe') -or
    $devExecutable -match '\\go-build[^\\]+\\[^\\]+\\exe\\server.exe$'
  )
  if (-not ($devManagedBinary -or $devLegacyBinary)) {
    throw "Port $devHttpPort is held by an unrelated process (PID $devServerOwner). Nothing was stopped."
  }
  $devBuildInfo = & $devGoExecutable version -m $devExecutable
  if ($LASTEXITCODE -ne 0 -or
      -not ($devBuildInfo -match 'path\s+github.com/jsness/recipe-extractor/server/cmd/server\s*$')) {
    throw "Cannot identify PID $devServerOwner as the recipe-extractor backend. Nothing was stopped."
  }
}

$env:DATABASE_URL = if ($env:DEV_DATABASE_URL) {
  $env:DEV_DATABASE_URL
} else {
  "postgres://postgres:postgres@localhost:$devPostgresPort/recipes?sslmode=disable"
}
Write-Host "Using isolated dev database on port $devPostgresPort."

$env:HTTP_ADDR = $devHttpAddr
$env:FRONTEND_DEV_PROXY_URL = "http://localhost:$devVitePort"
$env:VITE_DEV_PORT = $devVitePort
$env:VITE_API_PROXY_TARGET = "http://localhost$devHttpAddr"

Push-Location $root
try {
  Write-Host "Starting isolated dev Postgres container on localhost:$devPostgresPort..."
  docker compose -p $devComposeProject -f $devComposeFile up -d postgres

  Write-Host 'Waiting for Postgres to become healthy...'
  Wait-ForPostgres
} finally {
  Pop-Location
}

foreach ($devServerOwner in $devServerOwners) {
  Write-Host "Stopping previous dev Go server (PID $devServerOwner) on port $devHttpPort..."
  Stop-Process -Id $devServerOwner
}

Write-Host "Starting updated Go server on $devHttpAddr..."
$devStdout = Join-Path $devRuntimeDir 'server.stdout.log'
$devStderr = Join-Path $devRuntimeDir 'server.stderr.log'
$devServer = Start-Process -FilePath $devServerBinary -WorkingDirectory (Join-Path $root 'server') `
  -WindowStyle Hidden -RedirectStandardOutput $devStdout -RedirectStandardError $devStderr -PassThru
$devStartupDeadline = (Get-Date).AddSeconds(15)
do {
  $devServer.Refresh()
  if ($devServer.HasExited) {
    throw "Go server exited during startup. Check $devStderr and $devStdout."
  }
  $devListening = @(Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
    Where-Object { $_.LocalPort -eq $devHttpPort -and $_.OwningProcess -eq $devServer.Id })
  if ($devListening.Count -gt 0) { break }
  Start-Sleep -Milliseconds 250
} while ((Get-Date) -lt $devStartupDeadline)
if ($devListening.Count -eq 0) {
  throw "Go server did not listen on port $devHttpPort. Check $devStderr and $devStdout."
}
Write-Host "Go server ready (PID $($devServer.Id)). Logs: $devRuntimeDir"

# Vite reloads frontend edits automatically. Reuse it instead of starting duplicates.
$devViteOwners = @(Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
  Where-Object { $_.LocalPort -eq [int]$devVitePort } |
  Select-Object -ExpandProperty OwningProcess -Unique)
if ($devViteOwners.Count -gt 0) {
  foreach ($devViteOwner in $devViteOwners) {
    $devViteProcess = Get-CimInstance Win32_Process -Filter "ProcessId = $devViteOwner"
    if ($devViteProcess.CommandLine -notlike "*$root\web\node_modules*" -or
        $devViteProcess.CommandLine -notmatch 'vite') {
      throw "Port $devVitePort is held by an unrelated process. The Go server is ready; Vite was not started."
    }
  }
  Write-Host "Reusing Vite on port $devVitePort. Open http://localhost:$devHttpPort"
  return
}

$webCommand = @"
`$env:VITE_DEV_PORT = '$($env:VITE_DEV_PORT)';
`$env:VITE_API_PROXY_TARGET = '$($env:VITE_API_PROXY_TARGET)';
Set-Location '$root\web';
npm run dev
"@

Write-Host 'Starting Vite dev server...'
Start-Process -FilePath powershell -WindowStyle Hidden -ArgumentList @(
  '-NoProfile',
  '-Command',
  $webCommand
)
Write-Host "Open http://localhost:$devHttpPort"
