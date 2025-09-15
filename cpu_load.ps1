while ($true) {
    1..1000000 | ForEach-Object { $null }
    Write-Host "Generating CPU load..."
    Start-Sleep -Milliseconds 100
}