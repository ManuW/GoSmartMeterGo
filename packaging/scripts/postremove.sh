#!/bin/sh
set -e

# Disable Nginx proxy
if [ -f /etc/nginx/sites-enabled/gosmartmeter ]; then
    rm /etc/nginx/sites-enabled/gosmartmeter
fi

if [ -d /etc/nginx ] && nginx -t >/dev/null 2>&1; then
    systemctl reload nginx || systemctl restart nginx
fi

systemctl daemon-reload

if [ "$1" = "purge" ]; then
    # Remove config and database if user purged the package
    rm -rf /etc/gosmartmeter
    rm -rf /var/lib/gosmartmeter
fi
