#!/bin/bash
set -euo pipefail

PROJECT_DIR="/home/marklindgreen/multica-fork"
NODE_BIN="/home/marklindgreen/.nvm/versions/node/v22.22.2/bin"
DAEMON_USER="marklindgreen"
NEXT_DIST_DIR=".next-deploy"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"

export PATH="$PATH:$NODE_BIN"
cd "$PROJECT_DIR"
set -a
source "$PROJECT_DIR/.env"
set +a

echo "Building frontend..."
NEXT_DIST_DIR="$NEXT_DIST_DIR" pnpm run build

echo "Building backend binaries..."
cd "$PROJECT_DIR/server"
/usr/local/go/bin/go build -o bin/server ./cmd/server
/usr/local/go/bin/go build -o bin/multica ./cmd/multica
/usr/local/go/bin/go build -o bin/migrate ./cmd/migrate
cd "$PROJECT_DIR"

echo "Restarting NextJS..."
sudo -n pkill -f "next-server" || true
PORT="$FRONTEND_PORT" NEXT_DIST_DIR="$NEXT_DIST_DIR" nohup pnpm --filter @multica/web start > frontend.log 2>&1 &

echo "Waiting for frontend..."
for _ in $(seq 1 30); do
    if curl -fsS "http://127.0.0.1:${FRONTEND_PORT}" >/dev/null 2>&1; then
        echo "Frontend is healthy on :${FRONTEND_PORT}"
        break
    fi
    sleep 1
done

if ! curl -fsS "http://127.0.0.1:${FRONTEND_PORT}" >/dev/null 2>&1; then
    echo "Frontend failed to start"
    tail -n 100 frontend.log || true
    exit 1
fi

echo "Restarting Backend..."
sudo -n pkill -f "server/bin/server" || true
nohup ./server/bin/server > backend.log 2>&1 &

echo "Restarting Daemon..."
pkill -f "multica daemon" || true
export MULTICA_APP_URL="https://multica.zed.md"
export MULTICA_SERVER_URL="wss://multica.zed.md/ws"
nohup ./server/bin/multica daemon start >> /home/"$DAEMON_USER"/.multica/daemon.log 2>&1 &

echo "Done."
