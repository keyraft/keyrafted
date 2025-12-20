#!/bin/sh
set -e

# Data directory
DATA_DIR="${KEYRAFT_DATA_DIR:-/data}"
TOKEN_FILE="${DATA_DIR}/keyraft.db"

# Check if database exists, if not, initialize
if [ ! -f "$TOKEN_FILE" ]; then
    echo "🔧 Initializing Keyraft..."
    /app/keyrafted init --data-dir "$DATA_DIR"
    echo "✅ Initialization complete!"
    echo ""
fi

# Start the server
echo "🚀 Starting Keyraft server..."
exec /app/keyrafted start --data-dir "$DATA_DIR" "$@"

