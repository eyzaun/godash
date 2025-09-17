param(
    [string]$VersionInfo = "versioninfo.json",
    [string]$Icon = "web/static/favicon.ico"
)

# Ensures goversioninfo is available and generates resource.syso for Windows icon/version metadata
$ErrorActionPreference = "Stop"

# Install goversioninfo if missing
$goversioninfo = (Get-Command goversioninfo -ErrorAction SilentlyContinue)
if (-not $goversioninfo) {
    Write-Host "Installing goversioninfo..."
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
}

# Ensure icon path in versioninfo.json if provided via -Icon
if (Test-Path $VersionInfo) {
    try {
        $json = Get-Content $VersionInfo -Raw | ConvertFrom-Json
        if (-not $json.IconPath -and $Icon) {
            $json | Add-Member -NotePropertyName IconPath -NotePropertyValue $Icon
            ($json | ConvertTo-Json -Depth 10) | Set-Content $VersionInfo -NoNewline
            Write-Host "Updated IconPath in $VersionInfo -> $Icon"
        }
    } catch {
        Write-Warning "Could not parse $VersionInfo as JSON. Skipping IconPath auto-set."
    }
}

Write-Host "Generating resource.syso from $VersionInfo..."
# Rely on IconPath in versioninfo.json; avoid passing -icon to prevent variable literal issues
goversioninfo -o resource.syso -manifest= -64 $VersionInfo
Write-Host "resource.syso generated."