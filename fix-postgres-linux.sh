#!/bin/bash

echo "========================================"
echo "    PostgreSQL Linux Setup Fix"
echo "========================================"
echo

echo "🔧 This script will fix common PostgreSQL issues on Linux..."
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (use sudo)"
    echo "Run: sudo ./fix-postgres-linux.sh"
    exit 1
fi

echo "✅ Running as root"
echo

# Step 1: Start PostgreSQL service
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

# Step 2: Fix postgres user
echo "👤 Setting up postgres user..."
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" 2>/dev/null
sudo -u postgres psql -c "ALTER USER postgres CREATEDB;" 2>/dev/null
echo "✅ Postgres user configured"

echo

# Step 3: Create database
echo "🗄️  Creating mulic2_db database..."
if sudo -u postgres createdb mulic2_db 2>/dev/null; then
    echo "✅ Database 'mulic2_db' created successfully"
else
    echo "⚠️  Database may already exist or creation failed"
fi

echo

# Step 4: Test connection
echo "🧪 Testing database connection..."
if psql -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
    echo "✅ Database connection successful!"
    echo
    echo "🎉 PostgreSQL is now ready for MuliC2!"
    echo "💡 You can now run: ./run-mulic2.sh"
else
    echo "❌ Database connection still failed"
    echo
    echo "🔧 Manual steps to fix:"
    echo "1. Edit PostgreSQL config:"
    echo "   sudo nano /etc/postgresql/*/main/pg_hba.conf"
    echo
    echo "2. Change this line:"
    echo "   local all all peer"
    echo "   To:"
    echo "   local all all trust"
    echo
    echo "3. Restart PostgreSQL:"
    echo "   sudo systemctl restart postgresql"
    echo
    echo "4. Try running this script again"
fi

echo
echo "========================================"
