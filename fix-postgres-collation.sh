#!/bin/bash

echo "========================================"
echo "  PostgreSQL Collation Fix Script"
echo "========================================"
echo

echo "🔧 This script will fix PostgreSQL collation version mismatch..."
echo "💡 This is a common issue on Kali Linux and other rolling distributions"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (use sudo)"
    echo "Run: sudo ./fix-postgres-collation.sh"
    exit 1
fi

echo "✅ Running as root"
echo

# Step 1: Stop PostgreSQL
echo "🛑 Stopping PostgreSQL service..."
systemctl stop postgresql
echo "✅ PostgreSQL stopped"
echo

# Step 2: Backup current data (optional)
echo "💾 Creating backup of current PostgreSQL data..."
BACKUP_DIR="/tmp/postgresql_backup_$(date +%Y%m%d_%H%M%S)"
if [ -d "/var/lib/postgresql" ]; then
    mkdir -p "$BACKUP_DIR"
    cp -r /var/lib/postgresql/* "$BACKUP_DIR/" 2>/dev/null
    echo "✅ Backup created at: $BACKUP_DIR"
else
    echo "⚠️  No existing PostgreSQL data found to backup"
fi
echo

# Step 3: Remove old PostgreSQL data
echo "🗑️  Removing old PostgreSQL data..."
if [ -d "/var/lib/postgresql" ]; then
    rm -rf /var/lib/postgresql/*
    echo "✅ Old data removed"
else
    echo "⚠️  No PostgreSQL data directory found"
fi
echo

# Step 4: Reinitialize PostgreSQL
echo "🔄 Reinitializing PostgreSQL database..."

# Find PostgreSQL version and data directory
PG_VERSION=""
PG_DATA_DIR=""

# Try to find PostgreSQL version
if command -v postgres &> /dev/null; then
    PG_VERSION=$(postgres --version | grep -oE '[0-9]+' | head -1)
    echo "Found PostgreSQL version: $PG_VERSION"
fi

# Try different reinit methods
if command -v postgresql-setup &> /dev/null; then
    echo "Using postgresql-setup..."
    postgresql-setup --initdb
elif command -v pg_ctlcluster &> /dev/null; then
    echo "Using pg_ctlcluster..."
    # Find existing clusters
    CLUSTERS=$(pg_lsclusters | grep -v "Ver" | awk '{print $1 " " $2}' | head -1)
    if [ -n "$CLUSTERS" ]; then
        read PG_VERSION PG_CLUSTER <<< "$CLUSTERS"
        echo "Found cluster: $PG_VERSION $PG_CLUSTER"
        sudo -u postgres initdb -D "/var/lib/postgresql/$PG_VERSION/$PG_CLUSTER"
    else
        echo "No clusters found, trying default locations..."
        for version in 15 14 13 12; do
            if [ -d "/var/lib/postgresql/$version" ]; then
                echo "Found data directory: /var/lib/postgresql/$version"
                sudo -u postgres initdb -D "/var/lib/postgresql/$version/main"
                break
            fi
        done
    fi
else
    echo "❌ Cannot find PostgreSQL setup tools"
    echo "Trying alternative method..."
    
    # Try to find postgres user home and data directories
    POSTGRES_HOME=$(getent passwd postgres | cut -d: -f6)
    if [ -n "$POSTGRES_HOME" ]; then
        echo "Found postgres home: $POSTGRES_HOME"
        # Look for data directories
        for dir in "$POSTGRES_HOME"/*/main; do
            if [ -d "$dir" ]; then
                echo "Found data directory: $dir"
                sudo -u postgres initdb -D "$dir"
                break
            fi
        done
    fi
    
    # Try system data directories
    for version in 15 14 13 12; do
        if [ -d "/var/lib/postgresql/$version" ]; then
            echo "Found system data directory: /var/lib/postgresql/$version"
            sudo -u postgres initdb -D "/var/lib/postgresql/$version/main"
            break
        fi
    done
fi
echo

# Step 5: Start PostgreSQL
echo "🚀 Starting PostgreSQL service..."
systemctl start postgresql
systemctl enable postgresql

if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL service is running"
else
    echo "❌ Failed to start PostgreSQL service"
    exit 1
fi

echo

# Step 6: Wait for service to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5

# Step 7: Fix postgres user
echo "👤 Setting up postgres user..."
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" 2>/dev/null
sudo -u postgres psql -c "ALTER USER postgres CREATEDB;" 2>/dev/null
echo "✅ Postgres user configured"

echo

# Step 8: Create database
echo "🗄️  Creating mulic2_db database..."
if sudo -u postgres createdb mulic2_db 2>/dev/null; then
    echo "✅ Database 'mulic2_db' created successfully"
else
    echo "❌ Failed to create database"
    echo "Trying alternative method..."
    
    if sudo -u postgres psql -c "CREATE DATABASE mulic2_db;" 2>/dev/null; then
        echo "✅ Database 'mulic2_db' created successfully (SQL method)"
    else
        echo "❌ All database creation methods failed"
        exit 1
    fi
fi

echo

# Step 9: Test connection
echo "🧪 Testing database connection..."
if psql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
    echo "✅ Database connection successful!"
    echo
    echo "🎉 PostgreSQL collation issue is fixed!"
    echo "💡 You can now run: ./run-mulic2.sh"
else
    echo "❌ Database connection still failed"
    echo
    echo "🔧 Manual steps to fix:"
    echo "1. Completely reinstall PostgreSQL:"
    echo "   sudo apt remove --purge postgresql*"
    echo "   sudo apt autoremove"
    echo "   sudo apt install postgresql postgresql-contrib"
    echo
    echo "2. Or try the fix-postgres-linux.sh script first"
fi

echo
echo "========================================"
