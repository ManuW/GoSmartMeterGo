#!/bin/sh
set -e

if [ "$1" = "remove" ] || [ "$1" = "purge" ]; then
    if systemctl is-active --quiet gosmartmeter.service; then
        systemctl stop gosmartmeter.service
    fi
    systemctl disable gosmartmeter.service
fi
