#!/bin/bash
# Generate .env file with PROJECT_NAME and GIT_DIR for worktree support
# Runs automatically via devcontainer initializeCommand

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_NAME="$(basename "$PROJECT_DIR")"
# Write .env to project root (docker-compose looks there for variable substitution)
ENV_FILE="$PROJECT_DIR/.env"

# Check if this is a git worktree (has .git file instead of .git directory)
GIT_PATH="$PROJECT_DIR/.git"
WORKTREE_DIR=""
if [ -f "$GIT_PATH" ]; then
    # It's a worktree - extract the main .git directory
    GITDIR=$(cat "$GIT_PATH" | sed 's/gitdir: //')
    # Get the main repo's .git directory (parent of worktrees/)
    MAIN_GIT_DIR=$(dirname "$(dirname "$GITDIR")")
    # Set worktree dir for mounting at same host path
    WORKTREE_DIR="$PROJECT_DIR"
else
    # Regular repo - .git is the directory itself
    MAIN_GIT_DIR="$GIT_PATH"
fi

# Generate .env file
cat > "$ENV_FILE" << EOF
# Auto-generated - delete and re-run setup-env.sh to regenerate
PROJECT_NAME=$PROJECT_NAME
MAIN_GIT_DIR=$MAIN_GIT_DIR
WORKTREE_DIR=$WORKTREE_DIR
EOF

echo "Generated $ENV_FILE"
echo "  PROJECT_NAME=$PROJECT_NAME"
echo "  MAIN_GIT_DIR=$MAIN_GIT_DIR"
echo "  WORKTREE_DIR=$WORKTREE_DIR"
echo ""
echo "Endpoints:"
echo "  Frontend: http://$PROJECT_NAME.dev.local:3000"
echo "  Backend:  http://$PROJECT_NAME.dev.local:9999"
