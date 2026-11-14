#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || -z "${1:-}" ]]; then
    echo "Usage: $0 <user@host>" >&2
    echo "Example: $0 pi@ir.local" >&2
    exit 1
fi

TARGET="$1"
SERVICE_NAME="simrmt"
REMOTE_BIN="/usr/local/bin/${SERVICE_NAME}"
REMOTE_SERVICE="/etc/systemd/system/${SERVICE_NAME}.service"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

BIN_PATH="${TMP_DIR}/${SERVICE_NAME}"
SERVICE_PATH="${TMP_DIR}/${SERVICE_NAME}.service"

echo "[1/4] Building ${SERVICE_NAME} for linux/arm (32-bit)..."
(
    cd "${SCRIPT_DIR}"
    GOOS=linux GOARCH=arm go build -o "${BIN_PATH}" .
)

cat > "${SERVICE_PATH}" <<EOF
[Unit]
Description=simrmt LG WebOS TV IR Remote Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${REMOTE_BIN} -store /var/lib/${SERVICE_NAME}/store.json
StateDirectory=${SERVICE_NAME}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "[2/4] Copying binary and systemd unit to ${TARGET}..."
scp "${BIN_PATH}" "${SERVICE_PATH}" "${TARGET}:/tmp/"

echo "[3/4] Installing and enabling ${SERVICE_NAME}.service on ${TARGET}..."
ssh -t "${TARGET}" "
    set -euo pipefail
    sudo systemctl stop ${SERVICE_NAME}.service || true
    sudo install -m 0755 /tmp/${SERVICE_NAME} ${REMOTE_BIN}
    sudo install -m 0644 /tmp/${SERVICE_NAME}.service ${REMOTE_SERVICE}
    rm -f /tmp/${SERVICE_NAME} /tmp/${SERVICE_NAME}.service
    sudo systemctl daemon-reload
    sudo systemctl enable ${SERVICE_NAME}.service
    sudo systemctl restart ${SERVICE_NAME}.service
"

echo "[4/4] Checking service status..."
ssh "${TARGET}" "sudo systemctl status --no-pager ${SERVICE_NAME}.service"
