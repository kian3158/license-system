#!/usr/bin/env bash
set -euo pipefail

# Resolve script dir (project root)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Path to compiled client (adjust if your build outputs elsewhere)
CLIENT_PATH="$SCRIPT_DIR/client.dist/client"   # or "$SCRIPT_DIR/client.exe" on Windows builds

if [ ! -x "$CLIENT_PATH" ]; then
  echo "Client binary not found or not executable: $CLIENT_PATH"
  echo "Try building with Nuitka and ensure the output path is client.dist/client"
  exit 2
fi

# Optionally set env vars for runtime
export LICENSE_MANAGER_URL="${LICENSE_MANAGER_URL:-http://localhost:8080}"
export HW_EMULATOR_URL="${HW_EMULATOR_URL:-http://localhost:8000}"

# Run from repo root (so relative paths resolve consistently)
cd "$SCRIPT_DIR"
exec "$CLIENT_PATH" "$@"
