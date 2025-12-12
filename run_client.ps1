param(
    [string[]]$Args
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$ClientPath = Join-Path $ScriptDir "client.dist\client.exe"  # adjust if needed

if (-not (Test-Path $ClientPath)) {
    Write-Error "Client binary not found: $ClientPath"
    exit 2
}

# default envs
if (-not $env:LICENSE_MANAGER_URL) { $env:LICENSE_MANAGER_URL = "http://localhost:8080" }
if (-not $env:HW_EMULATOR_URL)    { $env:HW_EMULATOR_URL    = "http://localhost:8000" }

Push-Location $ScriptDir
& $ClientPath @Args
Pop-Location
