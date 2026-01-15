#!/bin/bash
# Generate .env file with PROJECT_NAME based on parent directory name
# Run this once per worktree before starting devpod

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_NAME="$(basename "$PROJECT_DIR")"
ENV_FILE="$SCRIPT_DIR/.env"

# Generate .env file
cat > "$ENV_FILE" << EOF
# Auto-generated from directory name: $PROJECT_NAME
# Delete this file and re-run setup-env.sh to regenerate
PROJECT_NAME=$PROJECT_NAME
EOF

echo "Generated $ENV_FILE with PROJECT_NAME=$PROJECT_NAME"
echo ""
echo "Endpoints will be:"
echo "  Frontend: http://$PROJECT_NAME.dev.local:3000"
echo "  Backend:  http://$PROJECT_NAME.dev.local:9999"
