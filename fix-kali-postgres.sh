#!/bin/bash

echo "🚀 Kali Linux PostgreSQL 17 Fix"
echo "========================================"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (use sudo)"
    echo "Run: sudo ./fix-kali-postgres.sh"
    exit 1
fi

echo "✅ Running as root"
echo

# Step 1: Stop PostgreSQL
echo "🛑 Stopping PostgreSQL..."
systemctl stop postgresql &> /dev/null
echo "✅ PostgreSQL stopped"
echo

# Step 2: Find PostgreSQL version and cluster
echo "🔍 Finding PostgreSQL setup..."
PG_VERSION=$(postgres --version 2>/dev/null | grep -oE '[0-9]+' | head -1)
if [ -z "$PG_VERSION" ]; then
    PG_VERSION="17"
fi
echo "✅ PostgreSQL version: $PG_VERSION"

# Find existing cluster
CLUSTER_INFO=$(pg_lsclusters 2>/dev/null | grep -v "Ver" | head -1)
if [ -n "$CLUSTER_INFO" ]; then
    read CLUSTER_VERSION CLUSTER_NAME <<< "$CLUSTER_INFO"
    echo "✅ Found cluster: $CLUSTER_VERSION $CLUSTER_NAME"
else
    echo "⚠️  No clusters found, will create new one"
    CLUSTER_VERSION="$PG_VERSION"
    CLUSTER_NAME="main"
fi
echo

# Step 3: Remove old data
echo "🗑️  Removing old PostgreSQL data..."
DATA_DIR="/var/lib/postgresql/$CLUSTER_VERSION/$CLUSTER_NAME"
if [ -d "$DATA_DIR" ]; then
    rm -rf "$DATA_DIR"/*
    echo "✅ Old data removed from $DATA_DIR"
else
    echo "⚠️  Data directory not found: $DATA_DIR"
fi
echo

# Step 4: Try different reinit methods
echo "🔄 Reinitializing PostgreSQL..."

# Method 1: Try postgresql-setup
if command -v postgresql-setup &> /dev/null; then
    echo "Using postgresql-setup..."
    postgresql-setup --initdb
    if [ $? -eq 0 ]; then
        echo "✅ PostgreSQL reinitialized with postgresql-setup"
        REINIT_SUCCESS=true
    else
        echo "❌ postgresql-setup failed"
        REINIT_SUCCESS=false
    fi
# Method 2: Try pg_ctlcluster
elif command -v pg_ctlcluster &> /dev/null; then
    echo "Using pg_ctlcluster..."
    # Drop the cluster first
    pg_dropcluster $CLUSTER_VERSION $CLUSTER_NAME &> /dev/null
    # Create new cluster
    pg_createcluster $CLUSTER_VERSION $CLUSTER_NAME
    if [ $? -eq 0 ]; then
        echo "✅ PostgreSQL reinitialized with pg_createcluster"
        REINIT_SUCCESS=true
    else
        echo "❌ pg_createcluster failed"
        REINIT_SUCCESS=false
    fi
# Method 3: Try manual initdb
elif command -v initdb &> /dev/null; then
    echo "Using initdb..."
    mkdir -p "$DATA_DIR"
    chown postgres:postgres "$DATA_DIR"
    sudo -u postgres initdb -D "$DATA_DIR"
    if [ $? -eq 0 ]; then
        echo "✅ PostgreSQL reinitialized with initdb"
        REINIT_SUCCESS=true
    else
        echo "❌ initdb failed"
        REINIT_SUCCESS=false
    fi
else
    echo "❌ No PostgreSQL setup tools found"
    echo "Trying to install missing tools..."
    
    # Try to install postgresql-common
    apt update &> /dev/null
    apt install -y postgresql-common &> /dev/null
    
    if command -v pg_createcluster &> /dev/null; then
        echo "✅ Installed pg_createcluster"
        pg_dropcluster $CLUSTER_VERSION $CLUSTER_NAME &> /dev/null
        pg_createcluster $CLUSTER_VERSION $CLUSTER_NAME
        if [ $? -eq 0 ]; then
            echo "✅ PostgreSQL reinitialized with pg_createcluster"
            REINIT_SUCCESS=true
        else
            REINIT_SUCCESS=false
        fi
    else
        echo "❌ Could not install PostgreSQL tools"
        REINIT_SUCCESS=false
    fi
fi

if [ "$REINIT_SUCCESS" != "true" ]; then
    echo "❌ All reinitialization methods failed"
    echo "💡 Please try: sudo apt install --reinstall postgresql postgresql-common"
    exit 1
fi

echo

# Step 5: Start PostgreSQL
echo "🚀 Starting PostgreSQL service..."
systemctl start postgresql
sleep 5

if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL service is running"
else
    echo "❌ Failed to start PostgreSQL service"
    exit 1
fi
echo

# Step 6: Wait for service to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 3

# Step 7: Setup user and database
echo "👤 Setting up postgres user..."
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" &> /dev/null
sudo -u postgres psql -c "ALTER USER postgres CREATEDB;" &> /dev/null
echo "✅ Postgres user configured"
echo

echo "🗄️  Creating mulic2_db database..."
if sudo -u postgres createdb mulic2_db &> /dev/null; then
    echo "✅ Database 'mulic2_db' created successfully"
else
    echo "❌ Failed to create database"
    echo "Trying SQL method..."
    if sudo -u postgres psql -c "CREATE DATABASE mulic2_db;" &> /dev/null; then
        echo "✅ Database 'mulic2_db' created successfully (SQL method)"
    else
        echo "❌ All database creation methods failed"
        exit 1
    fi
fi
echo

# Step 8: Test connection
echo "🧪 Testing database connection..."
if psql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
    echo "✅ Database connection successful!"
    echo
    echo "🎉 PostgreSQL is now fixed and ready!"
    echo "💡 You can now run: ./run-mulic2.sh"
else
    echo "❌ Database connection still failed"
    echo "💡 Please check PostgreSQL logs: journalctl -u postgresql"
fi

echo
echo "========================================"
