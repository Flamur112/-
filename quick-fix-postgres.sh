#!/bin/bash

echo "🚀 Quick PostgreSQL Fix for Kali Linux"
echo "========================================"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (use sudo)"
    echo "Run: sudo ./quick-fix-postgres.sh"
    exit 1
fi

echo "✅ Running as root"
echo

# Step 1: Stop PostgreSQL
echo "🛑 Stopping PostgreSQL..."
systemctl stop postgresql &> /dev/null
echo "✅ PostgreSQL stopped"
echo

# Step 2: Find and remove old data
echo "🗑️  Removing old PostgreSQL data..."
if [ -d "/var/lib/postgresql" ]; then
    rm -rf /var/lib/postgresql/*
    echo "✅ Old data removed"
else
    echo "⚠️  No PostgreSQL data directory found"
fi
echo

# Step 3: Find PostgreSQL version
echo "🔍 Finding PostgreSQL version..."
PG_VERSION=$(postgres --version 2>/dev/null | grep -oE '[0-9]+' | head -1)
if [ -n "$PG_VERSION" ]; then
    echo "✅ Found PostgreSQL version: $PG_VERSION"
else
    echo "⚠️  Could not determine version, trying common ones..."
    PG_VERSION="15"
fi
echo

# Step 4: Create data directory and reinitialize
echo "🔄 Reinitializing PostgreSQL..."
DATA_DIR="/var/lib/postgresql/$PG_VERSION/main"
mkdir -p "$DATA_DIR"
chown postgres:postgres "$DATA_DIR"

# Try to reinitialize
if sudo -u postgres initdb -D "$DATA_DIR" &> /dev/null; then
    echo "✅ PostgreSQL reinitialized successfully"
else
    echo "❌ Failed to reinitialize"
    echo "Trying alternative method..."
    
    # Try postgresql-setup
    if command -v postgresql-setup &> /dev/null; then
        postgresql-setup --initdb
        echo "✅ PostgreSQL reinitialized with postgresql-setup"
    else
        echo "❌ All reinit methods failed"
        exit 1
    fi
fi
echo

# Step 5: Start PostgreSQL
echo "🚀 Starting PostgreSQL service..."
systemctl start postgresql
sleep 3

if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL service is running"
else
    echo "❌ Failed to start PostgreSQL service"
    exit 1
fi
echo

# Step 6: Setup user and database
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
    exit 1
fi
echo

# Step 7: Test connection
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
