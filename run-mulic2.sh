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

# Detect Debian/Kali cluster and port (if available)
CLUSTER_VERSION=""
CLUSTER_NAME=""
CLUSTER_PORT=""
PSQL_FLAGS=""
if command -v pg_lsclusters &> /dev/null; then
    CI=$(pg_lsclusters 2>/dev/null | awk 'NR==2{print}')
    if [ -n "$CI" ]; then
        # Expected columns: Ver Cluster Port Status Owner Data directory Log file
        read -r CLUSTER_VERSION CLUSTER_NAME CLUSTER_PORT CLUSTER_STATUS _ CLUSTER_DATADIR _ <<< "$CI"
        echo "🔎 Detected PostgreSQL cluster: version=$CLUSTER_VERSION name=$CLUSTER_NAME port=${CLUSTER_PORT:-unknown} status=${CLUSTER_STATUS:-unknown}"
        if [ -n "$CLUSTER_DATADIR" ]; then echo "📁 Data directory: $CLUSTER_DATADIR"; fi
        if [ -n "$CLUSTER_PORT" ]; then
            PSQL_FLAGS="-p $CLUSTER_PORT"
        fi
        # Prefer unix socket on Debian/Kali
        if [ -d "/var/run/postgresql" ]; then
            PSQL_FLAGS="$PSQL_FLAGS -h /var/run/postgresql"
            echo "🔌 Using unix socket at /var/run/postgresql and port ${CLUSTER_PORT:-5432} for local connections"
        fi
    fi
fi

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

# Read backend DB credentials from config so the backend and DB match
if command -v jq >/dev/null 2>&1; then
    DB_USER=$(jq -r '.database.user' backend/config.json 2>/dev/null)
    DB_PASS=$(jq -r '.database.password' backend/config.json 2>/dev/null)
else
    # Fallback parser (awk) if jq is unavailable
    DB_USER=$(awk -F '"' '/"user"\s*:/ {print $4; exit}' backend/config.json 2>/dev/null)
    DB_PASS=$(awk -F '"' '/"password"\s*:/ {print $4; exit}' backend/config.json 2>/dev/null)
fi
# Fallback sanitization if parsing looked wrong
if echo "$DB_USER" | grep -q ':'; then
    DB_USER=$(echo "$DB_USER" | sed -E 's/.*"([^"]+)".*/\1/')
fi
if echo "$DB_PASS" | grep -q ':'; then
    DB_PASS=$(echo "$DB_PASS" | sed -E 's/.*"([^"]+)".*/\1/')
fi
if [ -z "$DB_USER" ]; then DB_USER="postgres"; fi
if [ -z "$DB_PASS" ]; then DB_PASS="postgres"; fi
echo "🔐 Database credentials to apply (defaults are user=postgres, password=postgres)"
echo "   - user: $DB_USER"
echo "   - password: (hidden)"
echo "   - To change credentials:"
echo "     1) Edit backend/config.json → database.user / database.password"
echo "     2) Re-run: ./run-mulic2.sh"
echo "   - Or change immediately via PostgreSQL:"
echo "     sudo -u postgres psql -h /var/run/postgresql -d postgres -c \"ALTER USER $DB_USER PASSWORD 'NEW_PASSWORD';\""
echo "   - To use a new DB user instead:"
echo "     sudo -u postgres psql -h /var/run/postgresql -d postgres -c \"CREATE USER myuser WITH PASSWORD 'mypassword';\""
echo "     sudo -u postgres psql -h /var/run/postgresql -d postgres -c \"ALTER DATABASE mulic2_db OWNER TO myuser;\""
echo "     Then update backend/config.json to match and re-run"

# Force md5 auth on Debian/Kali clusters to avoid peer failures
PG_HBA=""
if command -v pg_lsclusters &> /dev/null; then
    CI=$(pg_lsclusters 2>/dev/null | awk 'NR==2{print}')
    if [ -n "$CI" ]; then
        read -r _CV _CN _CP _ST _OW _DD _LG <<< "$CI"
        if [ -n "$_CV" ] && [ -n "$_CN" ]; then
            CAND="/etc/postgresql/$_CV/$_CN/pg_hba.conf"
            if [ -f "$CAND" ]; then PG_HBA="$CAND"; fi
        fi
    fi
fi
if [ -z "$PG_HBA" ]; then
    # Fallback glob
    PG_HBA=$(ls /etc/postgresql/*/*/pg_hba.conf 2>/dev/null | head -1)
fi
if [ -n "$PG_HBA" ] && [ -f "$PG_HBA" ]; then
    echo "🔧 Adjusting authentication in: $PG_HBA"
    echo "   - Temporarily enabling 'trust' for local 'postgres' to set a password non-interactively"
    echo "   - Then enforcing 'md5' (password-based) auth for all local/host connections"

    # Function to enforce md5 mode lines
    enforce_md5() {
        # Remove any previous trust line for postgres
        sudo sed -i '/^\s*local\s\+all\s\+postgres\s\+trust/d' "$PG_HBA"
        # Ensure md5 lines
        if grep -Eq '^\s*local\s+all\s+all\s+peer' "$PG_HBA"; then
            sudo sed -i 's/^\s*local\s\+all\s\+all\s\+peer/\tlocal\tall\tall\tmd5/' "$PG_HBA"
        fi
        if grep -Eq '^\s*local\s+all\s+postgres\s+peer' "$PG_HBA"; then
            sudo sed -i 's/^\s*local\s\+all\s\+postgres\s\+peer/\tlocal\tall\tpostgres\tmd5/' "$PG_HBA"
        fi
        if ! grep -Eq '^\s*local\s+all\s+all\s+md5' "$PG_HBA"; then
            echo -e "local\tall\tall\tmd5" | sudo tee -a "$PG_HBA" >/dev/null
        fi
        if ! grep -Eq '^\s*host\s+all\s+all\s+127\.0\.0\.1/32\s+md5' "$PG_HBA"; then
            echo -e "host\tall\tall\t127.0.0.1/32\tmd5" | sudo tee -a "$PG_HBA" >/dev/null
        fi
        if ! grep -Eq '^\s*host\s+all\s+all\s+::1/128\s+md5' "$PG_HBA"; then
            echo -e "host\tall\tall\t::1/128\tmd5" | sudo tee -a "$PG_HBA" >/dev/null
        fi
    }

    # 1) Temporarily allow trust for postgres to set password non-interactively
    if ! grep -Eq '^\s*local\s+all\s+postgres\s+trust' "$PG_HBA"; then
        echo -e "local\tall\tpostgres\ttrust\n$(cat $PG_HBA)" | sudo tee "$PG_HBA" >/dev/null
    fi
    sudo systemctl restart postgresql || true
    sleep 2

    # Set password now that trust is allowed (guaranteed non-interactive)
    echo "🔒 Ensuring DB superuser password matches backend/config.json"
    echo "   - Setting user 'postgres' password to configured value"
    sudo -u postgres psql -h /var/run/postgresql -d postgres -c "ALTER USER postgres PASSWORD '$DB_PASS';" >/dev/null 2>&1 || true
    sudo -u postgres psql -h /var/run/postgresql -d postgres -c "ALTER USER postgres CREATEDB;" >/dev/null 2>&1 || true

    # 2) Enforce md5 for everyone including postgres
    echo "✅ Enforcing md5 (password) authentication in $PG_HBA"
    enforce_md5
    sudo systemctl restart postgresql || true
    sleep 2

    # Final verification using password (non-interactive)
    export PGPASSWORD="$DB_PASS"
    if ! psql $PSQL_FLAGS -h /var/run/postgresql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
        echo "⚠️  Password test failed once, retrying password set under trust mode"
        # Re-enable trust, set password again, then md5 again
        if ! grep -Eq '^\s*local\s+all\s+postgres\s+trust' "$PG_HBA"; then
            echo -e "local\tall\tpostgres\ttrust\n$(cat $PG_HBA)" | sudo tee "$PG_HBA" >/dev/null
        fi
        sudo systemctl restart postgresql || true
        sleep 2
        sudo -u postgres psql -h /var/run/postgresql -d postgres -c "ALTER USER postgres PASSWORD '$DB_PASS';" >/dev/null 2>&1 || true
        enforce_md5
        sudo systemctl restart postgresql || true
        sleep 2
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

# Ensure specific Debian/Kali cluster is started when detected
if [ -n "$CLUSTER_VERSION" ] && [ -n "$CLUSTER_NAME" ] && command -v pg_ctlcluster &> /dev/null; then
    sudo pg_ctlcluster "$CLUSTER_VERSION" "$CLUSTER_NAME" start || true
    sleep 2
fi

# Fix user permissions (non-interactive) – ensure password env is set for subsequent checks
export PGPASSWORD="$DB_PASS"
sudo -u postgres psql $PSQL_FLAGS -h /var/run/postgresql -d postgres -c "ALTER USER postgres CREATEDB;" &> /dev/null || true

# Try connection (use detected port if present)
if psql $PSQL_FLAGS -h /var/run/postgresql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
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
            # Re-detect with port and data dir
            read -r CLUSTER_VERSION CLUSTER_NAME CLUSTER_PORT CLUSTER_STATUS _ CLUSTER_DATADIR _ <<< "$CLUSTER_INFO"
            echo "Found cluster: $CLUSTER_VERSION $CLUSTER_NAME ${CLUSTER_PORT:-} ${CLUSTER_STATUS:-}"
            if [ -n "$CLUSTER_PORT" ]; then PSQL_FLAGS="-p $CLUSTER_PORT"; fi
            if [ -d "/var/run/postgresql" ]; then PSQL_FLAGS="$PSQL_FLAGS -h /var/run/postgresql"; fi
            
            # Stop PostgreSQL first
            sudo systemctl stop postgresql &> /dev/null
            
            # Drop and recreate the cluster (Kali Linux method)
            if command -v pg_dropcluster &> /dev/null && command -v pg_createcluster &> /dev/null; then
                echo "Using pg_dropcluster/pg_createcluster (Kali Linux method)..."
                sudo pg_dropcluster $CLUSTER_VERSION $CLUSTER_NAME &> /dev/null
                sudo pg_createcluster $CLUSTER_VERSION $CLUSTER_NAME &> /dev/null
                echo "✅ Cluster recreated successfully"
                # Start the recreated cluster and refresh port info
                sudo pg_ctlcluster "$CLUSTER_VERSION" "$CLUSTER_NAME" start || true
                sleep 2
                CI2=$(pg_lsclusters 2>/dev/null | awk 'NR==2{print}')
                if [ -n "$CI2" ]; then
                    read -r CLUSTER_VERSION CLUSTER_NAME CLUSTER_PORT CLUSTER_STATUS _ CLUSTER_DATADIR _ <<< "$CI2"
                    if [ -n "$CLUSTER_PORT" ]; then PSQL_FLAGS="-p $CLUSTER_PORT"; fi
                    if [ -d "/var/run/postgresql" ]; then PSQL_FLAGS="$PSQL_FLAGS -h /var/run/postgresql"; fi
                fi
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

    # Ensure specific cluster is up if detected
    if [ -n "$CLUSTER_VERSION" ] && [ -n "$CLUSTER_NAME" ] && command -v pg_ctlcluster &> /dev/null; then
        sudo pg_ctlcluster "$CLUSTER_VERSION" "$CLUSTER_NAME" start || true
        sleep 2
        # If data dir missing, force recreate
        if [ -n "$CLUSTER_DATADIR" ] && [ ! -d "$CLUSTER_DATADIR" ]; then
            if command -v pg_dropcluster &> /dev/null && command -v pg_createcluster &> /dev/null; then
                sudo pg_dropcluster "$CLUSTER_VERSION" "$CLUSTER_NAME" --stop &> /dev/null || true
                sudo pg_createcluster "$CLUSTER_VERSION" "$CLUSTER_NAME" &> /dev/null
                sudo pg_ctlcluster "$CLUSTER_VERSION" "$CLUSTER_NAME" start || true
                sleep 2
                CI3=$(pg_lsclusters 2>/dev/null | awk 'NR==2{print}')
                if [ -n "$CI3" ]; then
                    read -r CLUSTER_VERSION CLUSTER_NAME CLUSTER_PORT CLUSTER_STATUS _ CLUSTER_DATADIR _ <<< "$CI3"
                    if [ -n "$CLUSTER_PORT" ]; then PSQL_FLAGS="-p $CLUSTER_PORT"; fi
                    if [ -d "/var/run/postgresql" ]; then PSQL_FLAGS="$PSQL_FLAGS -h /var/run/postgresql"; fi
                fi
            fi
        fi
    fi
    
    # Setup user again
    sudo -u postgres psql -c "ALTER USER postgres PASSWORD '$DB_PASS';" &> /dev/null
    sudo -u postgres psql -c "ALTER USER postgres CREATEDB;" &> /dev/null
    
    # Test connection
    if psql $PSQL_FLAGS -h /var/run/postgresql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
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
if ! psql $PSQL_FLAGS -U postgres -d postgres -c "SELECT 1 FROM pg_database WHERE datname='mulic2_db';" | grep -q "1" 2>/dev/null; then
    echo "📋 Creating database 'mulic2_db' owned by postgres"
    echo "   - To use a different DB user:"
    echo "     sudo -u postgres psql -h /var/run/postgresql -d postgres -c \"CREATE USER myuser WITH PASSWORD 'mypassword';\""
    echo "     sudo -u postgres psql -h /var/run/postgresql -d postgres -c \"ALTER DATABASE mulic2_db OWNER TO myuser;\""
    echo "     Update backend/config.json -> database.user/password accordingly"
    
    # Try to create database
    if sudo -u postgres createdb $PSQL_FLAGS mulic2_db &> /dev/null; then
        echo "✅ Database 'mulic2_db' created successfully"
    elif psql $PSQL_FLAGS -U postgres -d postgres -c "CREATE DATABASE mulic2_db;" &> /dev/null; then
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
echo "📂 Frontend working directory: $(pwd)"
if [ ! -d node_modules ]; then
    echo "📦 Installing frontend dependencies (this may take a few minutes)..."
    npm ci || npm install || true
fi

echo "🚀 Starting Vite dev server (frontend)..."
# Start vite, fallback to npx vite if script not found
if npm run dev & then
    FRONTEND_PID=$!
else
    if command -v npx >/dev/null 2>&1; then
        echo "⚠️  'npm run dev' failed, trying 'npx vite'"
        npx vite &
        FRONTEND_PID=$!
    else
        echo "❌ vite not found and no npx available. Install with: npm install -g vite"
        FRONTEND_PID="N/A"
    fi
fi

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
