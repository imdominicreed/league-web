#!/bin/bash
# Post-create setup script for dev container
# Note: Dotfiles (chezmoi, tmux, nvim) are handled by the base image entrypoint

set -e

# Fix Docker socket permissions - match container docker group GID to host socket GID
if [ -S /var/run/docker.sock ]; then
    SOCKET_GID=$(stat -c '%g' /var/run/docker.sock)
    CURRENT_GID=$(getent group docker | cut -d: -f3)
    if [ "$SOCKET_GID" != "$CURRENT_GID" ]; then
        echo "==> Fixing Docker group GID ($CURRENT_GID -> $SOCKET_GID)..."
        sudo groupmod -g "$SOCKET_GID" docker 2>/dev/null || true
    fi
fi

echo "==> Setting up project environment..."

# Set up PATH for Go, fnm, and local bin
export PATH="/usr/local/go/bin:$HOME/go/bin:$HOME/.local/share/fnm:$HOME/.local/bin:$PATH"

# Ensure nvm is loaded (fallback if fnm isn't available)
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

# Set up fnm if available (explicitly use bash output)
if command -v fnm &> /dev/null; then
    eval "$(fnm env --shell bash)"
fi

# Install Go dependencies
echo "==> Installing Go dependencies..."
go mod download

# Install frontend dependencies
echo "==> Installing frontend dependencies..."
cd frontend
npm install
cd ..

echo ""
echo "========================================"
echo "  Dev container ready!"
echo "========================================"
echo ""
echo "Database: postgres://postgres:postgres@db:5432/league_draft"
echo ""
echo "Commands:"
echo "  make dev-backend   - Start Go backend"
echo "  make dev-frontend  - Start React frontend"
echo "  make dev           - Start both (tmux)"
echo ""
