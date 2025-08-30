#!/bin/bash

echo "Checking PostgreSQL service status..."

# Check if PostgreSQL is running
if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL is running"
else
    echo "❌ PostgreSQL is not running"
    echo "Starting PostgreSQL service..."
    sudo systemctl start postgresql
    
    if systemctl is-active --quiet postgresql; then
        echo "✅ PostgreSQL started successfully"
    else
        echo "❌ Failed to start PostgreSQL"
        echo "Checking logs:"
        sudo journalctl -u postgresql --no-pager -n 20
        exit 1
    fi
fi

# Check if database exists
echo "Checking if mulic2_db exists..."
if sudo -u postgres psql -lqt | cut -d \| -f 1 | grep -qw mulic2_db; then
    echo "✅ Database mulic2_db exists"
else
    echo "❌ Database mulic2_db does not exist"
    echo "Creating database..."
    sudo -u postgres createdb mulic2_db
    echo "✅ Database created"
fi

# Test connection
echo "Testing database connection..."
if sudo -u postgres psql -d mulic2_db -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed"
    exit 1
fi

echo "Database check complete!"