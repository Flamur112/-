Write-Host "MuliC2 Environment Switcher" -ForegroundColor Cyan
Write-Host "===========================" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Local Development (localhost:8080)" -ForegroundColor Green
Write-Host "2. Linux C2 Server (192.168.0.111:8080)" -ForegroundColor Green
Write-Host "3. Custom Configuration" -ForegroundColor Green
Write-Host ""

$choice = Read-Host "Choose environment (1-3)"

switch ($choice) {
    "1" {
        Write-Host "Setting to localhost:8080..." -ForegroundColor Yellow
        $config = Get-Content 'config.json' | ConvertFrom-Json
        $config.backend.host = "localhost"
        $config.backend.api_port = 8080
        $config | ConvertTo-Json -Depth 10 | Set-Content 'config.json'
        Write-Host "Environment set to localhost:8080" -ForegroundColor Green
    }
    "2" {
        Write-Host "Setting to 192.168.0.111:8080..." -ForegroundColor Yellow
        $config = Get-Content 'config.json' | ConvertFrom-Json
        $config.backend.host = "192.168.0.111"
        $config.backend.api_port = 8080
        $config | ConvertTo-Json -Depth 10 | Set-Content 'config.json'
        Write-Host "Environment set to 192.168.0.111:8080" -ForegroundColor Green
    }
    "3" {
        $customHost = Read-Host "Enter custom host/IP"
        $customPort = Read-Host "Enter custom port (default 8080)"
        if ([string]::IsNullOrEmpty($customPort)) { $customPort = 8080 }
        
        Write-Host "Setting to $customHost`:$customPort..." -ForegroundColor Yellow
        $config = Get-Content 'config.json' | ConvertFrom-Json
        $config.backend.host = $customHost
        $config.backend.api_port = [int]$customPort
        $config | ConvertTo-Json -Depth 10 | Set-Content 'config.json'
        Write-Host "Environment set to $customHost`:$customPort" -ForegroundColor Green
    }
    default {
        Write-Host "Invalid choice. Please run the script again." -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "Current configuration:" -ForegroundColor Cyan
Get-Content 'config.json' | ConvertFrom-Json | ConvertTo-Json -Depth 10
Write-Host ""
Read-Host "Press Enter to continue"
