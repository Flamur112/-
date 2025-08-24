@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ========================================
echo           MuliC2 Launcher
echo ========================================
echo.

echo Checking prerequisites...
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH
    echo Please install Go from https://golang.org/
    echo.
    pause
    exit /b 1
)

REM Check if Node.js is installed
node --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js is not installed or not in PATH
    echo Please install Node.js from https://nodejs.org/
    echo.
    pause
    exit /b 1
)

echo [OK] Go and Node.js are installed
echo.

REM Ask user for PostgreSQL path
echo [INFO] Please specify the path to your PostgreSQL installation
echo.
echo Common locations:
echo   - C:\Program Files\PostgreSQL\17
echo   - C:\Program Files (x86)\PostgreSQL\17
echo   - T:\PostgreSQL\17
echo   - D:\PostgreSQL\17
echo   - Custom location...
echo.
set /p PG_PATH="Enter your PostgreSQL installation path (e.g., T:\PostgreSQL\17): "

if "%PG_PATH%"=="" (
    echo [ERROR] No path entered!
    echo.
    pause
    exit /b 1
)

REM Remove quotes if user entered them
set PG_PATH=%PG_PATH:"=%

REM Check if the path exists
if not exist "%PG_PATH%" (
    echo [ERROR] Path does not exist: %PG_PATH%
    echo Please check your PostgreSQL installation location.
    echo.
    pause
    exit /b 1
)

echo [OK] PostgreSQL path verified: %PG_PATH%
echo.

REM Set the paths
set PG_BIN=%PG_PATH%\bin
set PG_DATA=%PG_PATH%\data

REM Check if pg_ctl.exe exists
if not exist "%PG_BIN%\pg_ctl.exe" (
    echo [ERROR] pg_ctl.exe not found at: %PG_BIN%\pg_ctl.exe
    echo Please check if this is the correct PostgreSQL installation path.
    echo.
    pause
    exit /b 1
)

REM Check if data directory exists
if not exist "%PG_DATA%" (
    echo [ERROR] Data directory not found: %PG_DATA%
    echo Please check your PostgreSQL installation.
    echo.
    pause
    exit /b 1
)

echo [OK] PostgreSQL paths configured:
echo   PG_PATH: %PG_PATH%
echo   PG_BIN: %PG_BIN%
echo   PG_DATA: %PG_DATA%
echo.

REM Simple PostgreSQL check - just try to start it
echo [INFO] Attempting to start PostgreSQL...
cd /d "%PG_BIN%"
pg_ctl.exe start -D "%PG_DATA%" >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] PostgreSQL started successfully
) else (
    echo [WARN] pg_ctl failed, trying Windows service...
    sc start postgresql-x64-17 >nul 2>&1
    if %errorlevel% neq 0 (
        sc start postgresql >nul 2>&1
    )
)

echo [INFO] Waiting for PostgreSQL to start...
timeout /t 3 /nobreak >nul

REM Load password from config.json
echo [INFO] Loading database configuration...
if not exist "%~dp0backend\config.json" (
    echo [ERROR] backend\config.json not found at: %~dp0backend\config.json
    echo Please check if the file exists
    echo.
    pause
    exit /b 1
)

set PG_PASSWORD=
for /f "tokens=2 delims=:," %%a in ('findstr "password" "%~dp0backend\config.json"') do (
    set PG_PASSWORD=%%a
    set PG_PASSWORD=!PG_PASSWORD:"=!
    set PG_PASSWORD=!PG_PASSWORD: =!
)

if "%PG_PASSWORD%"=="" (
    echo [ERROR] Could not read password from backend\config.json
    echo Please check your configuration file
    echo.
    pause
    exit /b 1
)

echo [OK] Password loaded from config file
echo.

REM Check database connection
echo [INFO] Checking database connection...
set PGPASSWORD=%PG_PASSWORD%
"%PG_BIN%\psql.exe" -U postgres -d postgres -c "SELECT 1;" >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Cannot connect to PostgreSQL
    echo Please check your PostgreSQL password in backend\config.json
    echo.
    pause
    exit /b 1
)

echo [OK] PostgreSQL connection successful
echo.

REM Check if mulic2_db exists
echo [INFO] Checking if database 'mulic2_db' exists...
"%PG_BIN%\psql.exe" -U postgres -d postgres -c "SELECT 1 FROM pg_database WHERE datname='mulic2_db';" | findstr "1" >nul 2>&1
if %errorlevel% neq 0 (
    echo [INFO] Database 'mulic2_db' not found. Creating it now...
    "%PG_BIN%\psql.exe" -U postgres -d postgres -c "CREATE DATABASE mulic2_db;" >nul 2>&1
    if %errorlevel% neq 0 (
        echo [ERROR] Failed to create database
        echo Please check your PostgreSQL permissions
        echo.
        pause
        exit /b 1
    )
    echo [OK] Database 'mulic2_db' created successfully
) else (
    echo [OK] Database 'mulic2_db' already exists
)

REM Clear password from environment
set PGPASSWORD=

echo.
echo [INFO] Final verification...
echo [OK] PostgreSQL: Ready
echo [OK] Database: mulic2_db exists
echo [OK] Connection: Verified
echo [OK] Backend: Ready to start
echo [OK] Frontend: Ready to start
echo.
echo Starting MuliC2...
echo.

REM Start backend
echo [INFO] Starting backend server...
cd /d "%~dp0backend"

REM Check if mulic2.exe exists, if not build it
if not exist "mulic2.exe" (
    echo [INFO] Building backend executable...
    go build -o mulic2.exe
    if %errorlevel% neq 0 (
        echo [ERROR] Failed to build backend
        pause
        exit /b 1
    )
)

echo [INFO] Starting backend server...
start "MuliC2 Backend" cmd /c "mulic2.exe"

REM Wait longer for backend to fully start (including TLS validation)
echo [INFO] Waiting for backend to start and validate TLS certificates...
timeout /t 8 /nobreak >nul

REM Verify backend is running
echo [INFO] Verifying backend is running...
curl -s http://localhost:8080/api/health >nul 2>&1
if %errorlevel% neq 0 (
    echo [WARNING] Backend may not be fully started yet
    echo [INFO] Waiting additional time for backend...
    timeout /t 5 /nobreak >nul
)

REM Start frontend
echo [INFO] Starting frontend server...
cd /d "%~dp0frontend"
start "MuliC2 Frontend" cmd /c "npm run dev"

echo.
echo [OK] MuliC2 is starting up!
echo.
echo Backend: http://localhost:8080
echo Frontend: http://localhost:5173
echo.
echo Press any key to exit this launcher...
pause >nul
