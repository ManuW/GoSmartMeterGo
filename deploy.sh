#!/bin/bash
set -e

# Load variables from .env file if it exists
if [ -f .env ]; then
    source .env
fi

# Variables (defaults are overridden by .env or environment variables)
TARGET_USER="${TARGET_USER:-pi}"
TARGET_IP="${TARGET_IP:-pi.local}"
TARGET_DEST="${TARGET_DEST:-/tmp}"
DEB_FILE="gosmartmeter-rpi-32.deb"

echo "=== 1. Building and packaging for Raspberry Pi 32-bit (ARMv7) ==="
make pack-rpi-32

echo ""
echo "=== 2. Copying package via scp to ${TARGET_USER}@${TARGET_IP}:${TARGET_DEST}/ ==="
scp "${DEB_FILE}" "${TARGET_USER}@${TARGET_IP}:${TARGET_DEST}/"

echo ""
echo "=== 3. Installing package on Raspberry Pi ==="
# Using ssh -t to allocate a tty in case sudo prompts for a password
ssh -t "${TARGET_USER}@${TARGET_IP}" "sudo apt install -y ${TARGET_DEST}/${DEB_FILE}"

echo ""
echo "=== Deployment finished successfully! ==="
