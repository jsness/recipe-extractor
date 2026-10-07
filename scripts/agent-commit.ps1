param(
  [Parameter(Mandatory = $true)]
  [string]$MessageFile,
  [Parameter(Mandatory = $true)]
  [ValidatePattern('^[A-Za-z0-9._:/-]+(,[A-Za-z0-9._:/-]+)*$')]
  [string]$Model,
  [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9._-]*$')]
  [string]$OnBehalfOf = 'jsness',
  [switch]$CheckOnly
)

$ErrorActionPreference = 'Stop'
$message = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $MessageFile).Path).Trim()
if ([string]::IsNullOrWhiteSpace($message)) {
  throw 'Commit message must not be empty.'
}

# Git replaces existing values in the trailer block instead of duplicating them.
$formatted = $message | git interpret-trailers --if-exists replace `
  --trailer 'AI-Agent: OpenAI Codex' `
  --trailer "AI-Model: $Model" `
  --trailer "AI-On-Behalf-Of: $OnBehalfOf"
if ($LASTEXITCODE -ne 0) { throw 'Could not format commit trailers.' }

if ($CheckOnly) {
  $formatted
  return
}

$messagePath = [System.IO.Path]::GetTempFileName()
try {
  [System.IO.File]::WriteAllText($messagePath, ($formatted -join "`n") + "`n", [System.Text.UTF8Encoding]::new($false))
  git commit --file $messagePath
  if ($LASTEXITCODE -ne 0) { throw 'Git commit failed.' }
} finally {
  Remove-Item -LiteralPath $messagePath -Force
}
