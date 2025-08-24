#!/bin/bash

echo "========================================"
echo "           MuliC2 Launcher"
echo "========================================"
echo

echo "Checking prerequisites..."
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    echo "Please install Go from: https://golang.org/dl/"
    echo
    echo "Press Enter to exit..."
    read
    exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed or not in PATH"
    echo "Please install Node.js from: https://nodejs.org/"
    echo
    echo "Press Enter to exit..."
    read
    exit 1
fi

echo "✅ Go and Node.js are installed"
echo

# Check for PostgreSQL installation
echo "🔍 Checking PostgreSQL..."

# Check if PostgreSQL is installed
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL is not installed or not in PATH"
    echo "Please install PostgreSQL from: https://www.postgresql.org/download/"
    echo
    echo "Press Enter to exit..."
    read
    exit 1
fi

echo "✅ PostgreSQL found"
echo

# Check if PostgreSQL service is running
echo "🔍 Checking if PostgreSQL is running..."
if ! pg_isready -q; then
    echo "⚠️  PostgreSQL is not running. Starting it now..."
    
    # Try to start PostgreSQL (system-specific)
    if command -v systemctl &> /dev/null; then
        # Linux with systemd
        sudo systemctl start postgresql
    elif command -v brew &> /dev/null; then
        # macOS with Homebrew
        brew services start postgresql
    else
        echo "❌ Cannot start PostgreSQL automatically"
        echo "Please start PostgreSQL manually and try again"
        echo
        echo "Press Enter to exit..."
        read
        exit 1
    fi
    
    echo "⏳ Waiting for PostgreSQL to start..."
    sleep 5
fi

echo "✅ PostgreSQL is running"
echo

# Check if database exists and create if needed
echo "🔍 Checking database connection..."
if ! psql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
    echo "❌ Cannot connect to PostgreSQL"
    echo "Please check your PostgreSQL password and try again"
    echo
    echo "Press Enter to exit..."
    read
    exit 1
fi

echo "✅ PostgreSQL connection successful"
echo

# Check if mulic2_db exists
echo "🔍 Checking if database 'mulic2_db' exists..."
if ! psql -U postgres -d postgres -c "SELECT 1 FROM pg_database WHERE datname='mulic2_db';" | grep -q "1"; then
    echo "📋 Database 'mulic2_db' not found. Creating it now..."
    if ! psql -U postgres -d postgres -c "CREATE DATABASE mulic2_db;" &> /dev/null; then
        echo "❌ Failed to create database"
        echo "Please check your PostgreSQL permissions"
        echo
        echo "Press Enter to exit..."
        read
        exit 1
    fi
    echo "✅ Database 'mulic2_db' created successfully"
else
    echo "✅ Database 'mulic2_db' already exists"
fi

echo
echo "Starting MuliC2..."
echo

echo "🚀 Starting Backend Server..."
cd backend

# Check if mulic2 executable exists, if not build it
if [ ! -f "mulic2" ]; then
    echo "📦 Building backend executable..."
    go build -o mulic2
    if [ $? -ne 0 ]; then
        echo "❌ Failed to build backend"
        exit 1
    fi
fi

# Start backend in background
./mulic2 &
BACKEND_PID=$!

echo "⏳ Waiting for backend to start and validate TLS certificates..."
sleep 8

# Verify backend is running
echo "🔍 Verifying backend is running..."
if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "✅ Backend is running successfully"
else
    echo "⚠️  Backend may not be fully started yet"
    echo "⏳ Waiting additional time for backend..."
    sleep 5
fi

echo "🌐 Starting Frontend..."
cd ../frontend
npm run dev &
FRONTEND_PID=$!

echo
echo "========================================"
echo "✅ MuliC2 is starting up!"
echo
echo "📱 Frontend: http://localhost:5173"
echo "🔧 Backend API: http://localhost:8080"
echo "🎯 C2 Listener: Port 8443 (TLS encrypted)"
echo
echo "💡 Services are running in background"
echo "💡 Backend PID: $BACKEND_PID"
echo "💡 Frontend PID: $FRONTEND_PID"
echo "💡 To stop: kill $BACKEND_PID $FRONTEND_PID"
echo "========================================"
echo
echo "Launcher script completed. Services are running in background."
echo
echo "To stop all services:"
echo "kill $BACKEND_PID $FRONTEND_PID"
echo
echo "This window will close automatically in 10 seconds..."
sleep 10
