param([Parameter(Mandatory=$true)][string]$Version)
$ErrorActionPreference = 'Stop'
$env:GOWORK = 'off'
$env:CGO_ENABLED = '1'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$stage = Join-Path (Get-Location) 'build/windows-package'
New-Item -ItemType Directory -Force "$stage/cli", 'dist' | Out-Null
Copy-Item 'build/bin/SKM.exe' "$stage/SKM.exe"
Push-Location backend
try {
    go build -trimpath -ldflags "-s -w -X backend-go/app.Version=$Version" -o "$stage/cli/skm.exe" ./cmd/skm
    if ($LASTEXITCODE -ne 0) { throw 'CLI build failed' }
} finally { Pop-Location }
$actual = & "$stage/cli/skm.exe" version
if ($LASTEXITCODE -ne 0 -or $actual -ne $Version) { throw 'CLI version smoke test failed' }
# Run the packaged CLI against a fresh SQLite database outside the checkout.
$previousDSN = $env:DB_DSN
$previousDriver = $env:DB_DRIVER
$db = Join-Path ([System.IO.Path]::GetTempPath()) ("skm-release-" + [guid]::NewGuid() + '.db')
try {
    $env:DB_DRIVER = 'sqlite'
    $env:DB_DSN = $db
    & "$stage/cli/skm.exe" dashboard
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $db)) { throw 'Packaged CLI SQLite smoke test failed' }
} finally {
    $env:DB_DSN = $previousDSN
    $env:DB_DRIVER = $previousDriver
    Remove-Item "$db*" -ErrorAction SilentlyContinue
}
Copy-Item LICENSE "$stage/LICENSE.txt"
Copy-Item docs/release-notes.md "$stage/README.txt"
Compress-Archive -Path "$stage/*" -DestinationPath "dist/SKM-$Version-windows-x64.zip" -Force
