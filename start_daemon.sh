#!/bin/bash
set -euo pipefail

EMAIL="admin@zed.md"
API_URL="https://multica.zed.md/api"
PROJECT_DIR="/home/marklindgreen/multica-fork"
NODE_BIN="/home/marklindgreen/.nvm/versions/node/v22.22.2/bin"

echo "Sending code to $EMAIL..."
curl -s -X POST "$API_URL/auth/send-code" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\"}"

echo "Fetching code from DB..."
sleep 2
CODE=$(ssh -T boston "sudo docker exec multica-postgres-1 psql -U multica -d multica -t -c \"SELECT code FROM verification_code WHERE email = '$EMAIL' ORDER BY expires_at DESC LIMIT 1;\"" | tr -d ' ' | tr -d '\n')
echo "Found code: $CODE"

echo "Verifying code..."
RESP=$(curl -s -X POST "$API_URL/auth/verify-code" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"code\":\"$CODE\"}")

TOKEN=$(echo "$RESP" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo "Failed to get JWT token: $RESP"
    exit 1
fi
echo "Got JWT!"

echo "Generating PAT..."
PAT_RESP=$(curl -s -X POST "$API_URL/tokens" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Boston Daemon Token\"}")

PAT=$(echo "$PAT_RESP" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -z "$PAT" ]; then
    echo "Failed to get PAT: $PAT_RESP"
    exit 1
fi
echo "Generated PAT!"

echo "Logging in and starting daemon on Boston..."
ssh -T boston << EOF
set -euo pipefail
cd "$PROJECT_DIR"
export MULTICA_APP_URL=https://multica.zed.md
export MULTICA_SERVER_URL=wss://multica.zed.md/ws
export PATH="/usr/local/go/bin:$NODE_BIN:\$PATH"
export NVM_DIR="\$HOME/.nvm"
[ -s "\$NVM_DIR/nvm.sh" ] && \. "\$NVM_DIR/nvm.sh"
multica login --token "$PAT"
pkill -f "multica daemon" || true
nohup multica daemon start >> /home/marklindgreen/.multica/daemon.log 2>&1 &
echo "Daemon started successfully!"
EOF
