@echo off
echo MuliC2 Environment Switcher
echo ===========================
echo.
echo 1. Local Development (localhost:8080)
echo 2. Linux C2 Server (192.168.0.111:8080)
echo 3. Custom Configuration
echo.
set /p choice="Choose environment (1-3): "

if "%choice%"=="1" (
    echo Setting to localhost:8080...
    powershell -Command "(Get-Content 'config.json') -replace '\"host\": \".*\"', '\"host\": \"localhost\"' | Set-Content 'config.json'"
    echo Environment set to localhost:8080
) else if "%choice%"=="2" (
    echo Setting to 192.168.0.111:8080...
    powershell -Command "(Get-Content 'config.json') -replace '\"host\": \".*\"', '\"host\": \"192.168.0.111\"' | Set-Content 'config.json'"
    echo Environment set to 192.168.0.111:8080
) else if "%choice%"=="3" (
    set /p custom_host="Enter custom host/IP: "
    set /p custom_port="Enter custom port (default 8080): "
    if "%custom_port%"=="" set custom_port=8080
    echo Setting to %custom_host%:%custom_port%...
    powershell -Command "(Get-Content 'config.json') -replace '\"host\": \".*\"', '\"host\": \"%custom_host%\"' | Set-Content 'config.json'"
    powershell -Command "(Get-Content 'config.json') -replace '\"api_port\": .*', '\"api_port\": %custom_port%' | Set-Content 'config.json'"
    echo Environment set to %custom_host%:%custom_port%
) else (
    echo Invalid choice. Please run the script again.
)

echo.
echo Current configuration:
type config.json
echo.
pause
