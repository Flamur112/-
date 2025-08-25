#!/bin/bash

# Check if user wants help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "🚀 MuliC2 Linux Launcher"
    echo
    echo "Usage:"
    echo "  ./run-mulic2.sh          # Auto-fix everything and start MuliC2"
    echo "  ./run-mulic2.sh --help   # Show this help"
    echo
    echo "The script will automatically:"
    echo "  ✅ Start PostgreSQL automatically"
    echo "  ✅ Fix user permissions"
    echo "  ✅ Create database if needed"
    echo "  ✅ Start MuliC2 immediately"
    echo
    exit 0
fi

echo "🚀 Auto-fixing everything and starting MuliC2..."
echo "💡 This will automatically fix PostgreSQL issues"
echo

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

# Fix common PostgreSQL configuration issues on Linux
echo "🔧 Checking PostgreSQL configuration..."
if [ -f "/etc/postgresql/*/main/pg_hba.conf" ]; then
    echo "💡 PostgreSQL configuration found, checking authentication..."
    
    # Check if local connections are allowed
    if ! grep -q "local.*all.*all.*trust" /etc/postgresql/*/main/pg_hba.conf 2>/dev/null; then
        echo "⚠️  PostgreSQL may need authentication configuration"
        echo "💡 If you get connection errors, run:"
        echo "   sudo nano /etc/postgresql/*/main/pg_hba.conf"
        echo "   Change 'local all all peer' to 'local all all trust'"
        echo "   Then restart PostgreSQL: sudo systemctl restart postgresql"
        echo
    fi
fi

# Auto-fix PostgreSQL issues
echo "🔧 Auto-fixing PostgreSQL setup..."

# Start PostgreSQL if not running
if ! systemctl is-active --quiet postgresql; then
    echo "🚀 Starting PostgreSQL service..."
    sudo systemctl start postgresql
    sleep 2
fi

# Fix user permissions
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" &> /dev/null
sudo -u postgres psql -c "ALTER USER postgres CREATEDB;" &> /dev/null

# Try connection
if psql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
    echo "✅ PostgreSQL connection successful"
else
    echo "❌ PostgreSQL connection failed"
    echo "🔧 Auto-fixing collation issues..."
    
    # Stop PostgreSQL
    sudo systemctl stop postgresql &> /dev/null
    
    # Try to reinitialize if data directory exists
    if [ -d "/var/lib/postgresql" ] && [ "$(ls -A /var/lib/postgresql)" ]; then
        echo "🔄 Reinitializing PostgreSQL to fix collation..."
        
        # Find existing cluster
        CLUSTER_INFO=$(pg_lsclusters 2>/dev/null | grep -v "Ver" | head -1)
        if [ -n "$CLUSTER_INFO" ]; then
            read CLUSTER_VERSION CLUSTER_NAME <<< "$CLUSTER_INFO"
            echo "Found cluster: $CLUSTER_VERSION $CLUSTER_NAME"
            
            # Stop PostgreSQL first
            sudo systemctl stop postgresql &> /dev/null
            
            # Drop and recreate the cluster (Kali Linux method)
            if command -v pg_dropcluster &> /dev/null && command -v pg_createcluster &> /dev/null; then
                echo "Using pg_dropcluster/pg_createcluster (Kali Linux method)..."
                sudo pg_dropcluster $CLUSTER_VERSION $CLUSTER_NAME &> /dev/null
                sudo pg_createcluster $CLUSTER_VERSION $CLUSTER_NAME &> /dev/null
                echo "✅ Cluster recreated successfully"
            else
                echo "pg_dropcluster/pg_createcluster not found, trying alternative..."
                sudo rm -rf /var/lib/postgresql/* &> /dev/null
                
                # Try different reinit methods
                if command -v postgresql-setup &> /dev/null; then
                    echo "Using postgresql-setup..."
                    sudo postgresql-setup --initdb &> /dev/null
                else
                    echo "❌ No PostgreSQL setup tools found"
                    echo "💡 Please run: sudo ./fix-kali-postgres.sh"
                    echo
                    echo "Press Enter to exit..."
                    read
                    exit 1
                fi
            fi
        else
            echo "No clusters found, trying default reinit..."
            sudo rm -rf /var/lib/postgresql/* &> /dev/null
            
            if command -v postgresql-setup &> /dev/null; then
                sudo postgresql-setup --initdb &> /dev/null
            else
                echo "❌ No PostgreSQL setup tools found"
                echo "💡 Please run: sudo ./fix-kali-postgres.sh"
                echo
                echo "Press Enter to exit..."
                read
                exit 1
            fi
        fi
    fi
    
    # Start PostgreSQL
    sudo systemctl start postgresql
    sleep 3
    
    # Setup user again
    sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" &> /dev/null
    sudo -u postgres psql -c "ALTER USER postgres CREATEDB;" &> /dev/null
    
    # Test connection
    if psql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
        echo "✅ PostgreSQL fixed and connection successful"
    else
        echo "❌ PostgreSQL still not working"
        echo "💡 Please run: sudo ./fix-kali-postgres.sh (for Kali Linux)"
        echo "   or: sudo ./fix-postgres-collation.sh (for other Linux)"
        echo
        echo "Press Enter to exit..."
        read
        exit 1
    fi
fi

echo

# Auto-create database if needed
echo "🗄️  Checking database 'mulic2_db'..."
if ! psql -U postgres -d postgres -c "SELECT 1 FROM pg_database WHERE datname='mulic2_db';" | grep -q "1" 2>/dev/null; then
    echo "📋 Creating database 'mulic2_db'..."
    
    # Try to create database
    if sudo -u postgres createdb mulic2_db &> /dev/null; then
        echo "✅ Database 'mulic2_db' created successfully"
    elif psql -U postgres -d postgres -c "CREATE DATABASE mulic2_db;" &> /dev/null; then
        echo "✅ Database 'mulic2_db' created successfully (SQL method)"
    else
        echo "❌ Failed to create database"
        echo "💡 Please run: sudo ./fix-postgres-collation.sh"
        echo
        echo "Press Enter to exit..."
        read
        exit 1
    fi
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
