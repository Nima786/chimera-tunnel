#!/bin/bash
# This script is designed to be run via a curl | bash command.
# It installs and configures the Chimera client on a new server.
set -e

# --- Configuration ---
# The configuration JSON is passed as the first argument from the curl command.
if [ -z "$1" ]; then
    echo "[ERROR] Configuration JSON was not provided to the installer script. Aborting."
    exit 1
fi
CONFIG_JSON=$1

# Static variables for the installation
RELEASE_BASE_URL="https://github.com/Nima786/chimera-tunnel/releases/download/v0.3.0"
CHIMERA_BINARY_PATH="/usr/local/bin/chimera"
CHIMERA_CONFIG_DIR="/etc/chimera"
CLIENT_CONFIG_PATH="${CHIMERA_CONFIG_DIR}/client.json"
SERVICE_FILE_PATH="/etc/systemd/system/chimera-client.service"

# --- Main Installation Logic ---

echo "[INFO] Detecting system architecture..."
ARCH=$(uname -m)
BINARY_NAME=""
if [ "$ARCH" = "x86_64" ]; then
    BINARY_NAME="chimera-amd64"
elif [ "$ARCH" = "aarch64" ]; then
    BINARY_NAME="chimera-arm64"
else
    echo "[ERROR] This script does not support the '$ARCH' architecture."
    exit 1
fi
echo "[INFO] Detected $ARCH. Will download binary: $BINARY_NAME"

BINARY_URL="${RELEASE_BASE_URL}/${BINARY_NAME}"

echo "[INFO] Downloading and installing the Chimera binary..."
# Use curl to download the binary to its final destination and make it executable
curl -L -o ${CHIMERA_BINARY_PATH} "$BINARY_URL"
chmod +x ${CHIMERA_BINARY_PATH}

echo "[INFO] Creating configuration directory and file..."
mkdir -p ${CHIMERA_CONFIG_DIR}
echo "${CONFIG_JSON}" > ${CLIENT_CONFIG_PATH}

echo "[INFO] Creating systemd service file..."
cat <<EOF > ${SERVICE_FILE_PATH}

[Unit]
Description=Chimera Client Tunnel
After=network.target
[Service]
ExecStart=${CHIMERA_BINARY_PATH} -config ${CLIENT_CONFIG_PATH}
Restart=always
User=root
RestartSec=5
[Install]
WantedBy=multi-user.target
EOF

echo "[INFO] Enabling and starting the Chimera client service..."
systemctl daemon-reload
systemctl enable chimera-client.service
systemctl restart chimera-client.service

# Add a small delay to give systemd time to process the new unit
sleep 2

# Final verification check
if systemctl is-active --quiet chimera-client.service; then
    echo "[SUCCESS] Chimera client service is active and running!"
else
    echo "[ERROR] The Chimera client service failed to start. Please check the logs for errors:"
    echo "journalctl -u chimera-client.service"
    exit 1
fi
