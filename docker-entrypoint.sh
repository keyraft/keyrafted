#!/bin/sh
set -e

# Data directory
DATA_DIR="${KEYRAFT_DATA_DIR:-/data}"
TOKEN_FILE="${DATA_DIR}/keyraft.db"

# Ensure data directory exists
mkdir -p "$DATA_DIR"

# Check if database exists, if not, initialize
if [ ! -f "$TOKEN_FILE" ]; then
    echo "🔧 Initializing Keyraft..."
    if ! /app/keyrafted init --data-dir "$DATA_DIR"; then
        echo "❌ Failed to initialize Keyraft"
        exit 1
    fi
    echo "✅ Initialization complete!"
    echo ""
    
    # Verify database was created
    if [ ! -f "$TOKEN_FILE" ]; then
        echo "❌ Database file not created after initialization"
        exit 1
    fi
fi

# Start the server
echo "🚀 Starting Keyraft server..."
exec /app/keyrafted start --data-dir "$DATA_DIR" "$@"

