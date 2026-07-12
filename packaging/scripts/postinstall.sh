#!/bin/sh
set -e

# 1. Adjust permissions on config & state directories
if [ ! -f /etc/gosmartmeter/config.yaml ]; then
    echo "Creating default configuration file from example..."
    cp /etc/gosmartmeter/config.yaml.example /etc/gosmartmeter/config.yaml
fi
chown root:gosmartmeter /etc/gosmartmeter/config.yaml
chmod 640 /etc/gosmartmeter/config.yaml

mkdir -p /var/lib/gosmartmeter
chown -R gosmartmeter:gosmartmeter /var/lib/gosmartmeter
chmod 750 /var/lib/gosmartmeter

# 2. Generate SSL certificate if missing
if [ ! -f /etc/ssl/certs/gosmartmeter.crt ] || [ ! -f /etc/ssl/private/gosmartmeter.key ]; then
    echo "Generating self-signed SSL certificate..."
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
      -keyout /etc/ssl/private/gosmartmeter.key \
      -out /etc/ssl/certs/gosmartmeter.crt \
      -subj "/CN=gosmartmeter.local"
fi

# 3. Configure Nginx proxy if Nginx is installed
if [ -d /etc/nginx/sites-enabled ]; then
    # Disable default Nginx page
    if [ -f /etc/nginx/sites-enabled/default ]; then
        rm /etc/nginx/sites-enabled/default
    fi
    # Enable gosmartmeter site
    if [ ! -f /etc/nginx/sites-enabled/gosmartmeter ]; then
        ln -s /etc/nginx/sites-available/gosmartmeter /etc/nginx/sites-enabled/gosmartmeter
    fi
    # Test and reload Nginx
    if nginx -t >/dev/null 2>&1; then
        systemctl reload nginx || systemctl restart nginx
    else
        echo "Warning: Nginx configuration test failed!"
    fi
fi

# 4. Enable and start/restart systemd service
systemctl daemon-reload
systemctl enable gosmartmeter.service

if systemctl is-active --quiet gosmartmeter.service; then
    systemctl restart gosmartmeter.service
else
    systemctl start gosmartmeter.service
fi
