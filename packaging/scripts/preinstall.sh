#!/bin/sh
set -e

# Create gosmartmeter system user/group if they don't exist
if ! getent group gosmartmeter >/dev/null; then
    groupadd -r gosmartmeter
fi

if ! getent passwd gosmartmeter >/dev/null; then
    useradd -r -g gosmartmeter -d /var/lib/gosmartmeter -s /usr/sbin/nologin -c "GoSmartMeterGo System User" gosmartmeter
fi

# Ensure user is in dialout group to access /dev/ttyAMA0
usermod -aG dialout gosmartmeter
