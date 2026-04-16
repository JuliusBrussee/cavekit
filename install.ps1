Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinDir = Join-Path $env:LOCALAPPDATA "Programs\cavekit\bin"
$ExePath = Join-Path $BinDir "cavekit.exe"

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

Push-Location $RepoRoot
try {
  go build -o $ExePath .\cmd\cavekit
  & $ExePath install --source-dir $RepoRoot
} finally {
  Pop-Location
}
