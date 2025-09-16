param([int]$duration = 30)
$endTime = (Get-Date).AddSeconds($duration)
Write-Host "Starting CPU stress test for $duration seconds..."
while ((Get-Date) -lt $endTime) {
    1..100000 | ForEach-Object { $null = [math]::Sqrt($_) }
}
Write-Host "CPU stress test completed."