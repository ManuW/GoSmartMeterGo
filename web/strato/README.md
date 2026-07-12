# Strato Online-Viewer Deployment & Setup

Diese Anleitung beschreibt, wie das Web-Frontend und die API-Schnittstelle auf dem Strato-Server eingerichtet werden.

## Zielordner auf Strato
Alle Dateien aus diesem Verzeichnis (`web/strato/`) müssen in den folgenden Ordner auf deinem Strato-Webspace kopiert werden:
`example/www/`

## Dateien
*   `index.html` - Das Dashboard (HTML-Struktur).
*   `index.css` - Stylesheet für das Design.
*   `index.js` - Client-seitige Logik (Diagramme rendern, API aufrufen).
*   `api.php` - API-Brücke zur SQLite-Datenbank.
*   `.htaccess` - Verzeichnis- und Dateischutz.
*   `backups/` - Ordner, in den die SQLite-Backups via rsync hochgeladen werden.

---

## 1. Verzeichnisstruktur anlegen
Stelle sicher, dass im Ordner `example/www/` auf dem Strato-Server ein Unterordner namens `backups` existiert. In diesen wird die Go-App die Datenbanken sichern.
```bash
# Struktur auf Strato:
example/www/
├── index.html
├── index.css
├── index.js
├── api.php
├── .htaccess
└── backups/           <-- Diesen Ordner manuell erstellen (oder per FTP/SFTP)
```

---

## 2. Sicherheits-Setup (Zugriffsschutz)

### A. Schutz der `.db` Dateien vor direktem Download
Die `.htaccess` im Hauptordner (`example/www/`) verhindert, dass jemand deine SQLite-Dateien direkt im Browser herunterladen kann:
```apache
<Files ~ "\.db$">
    Order allow,deny
    Deny from all
</Files>
```
Dies schützt alle Dateien in `backups/` vor direktem Web-Zugriff, erlaubt dem lokalen PHP (`api.php`) aber weiterhin das Lesen der Datenbanken.

### B. Passwortschutz (HTTP Basic Auth) für das Dashboard
Um das gesamte Dashboard vor unbefugten Blicken zu schützen, empfehlen wir die Einrichtung von Basic Auth.

#### Option 1: Über den Strato Kundenbereich (Empfohlen)
1. Logge dich im Strato Kundenbereich ein.
2. Gehe zu **Sicherheit -> Verzeichnisschutz**.
3. Wähle das Verzeichnis `example/www` aus.
4. Lege einen Benutzernamen und ein Passwort fest und aktiviere den Schutz. Strato generiert die nötigen `.htaccess` und `.htpasswd` Dateien automatisch im Hintergrund.

#### Option 2: Manuell via `.htaccess` und `.htpasswd`
Erstelle eine `.htaccess` im Ordner `example/www/` mit folgendem Inhalt (Pfad anpassen!):
```apache
AuthType Basic
AuthName "SmartMeter Dashboard"
AuthUserFile /home/strato/http/power/rid/xx/xx/xxxxxxx/htdocs/example/www/.htpasswd
Require valid-user
```
Erstelle eine `.htpasswd` mit verschlüsseltem Passwort (z.B. über einen Online-Generator) und lade sie in denselben Ordner hoch.

---

## 3. SSH-Key für automatischen Upload einrichten (Raspberry Pi -> Strato)

Damit der Raspberry Pi per `rsync` ohne Passworteingabe auf den Strato-Server zugreifen kann:

1. **SSH-Key auf dem Raspberry Pi generieren** (falls noch nicht vorhanden):
   ```bash
   ssh-keygen -t ed25519
   ```
   *(Drücke dreimal Enter, um kein Passwort/Passphrase zu vergeben)*

2. **Public Key anzeigen:**
   ```bash
   cat ~/.ssh/id_ed25519.pub
   ```
   *Kopiere diese Zeile in die Zwischenablage.*

3. **Verzeichnis und Berechtigungen auf Strato anlegen:**
   Logge dich per SSH bei Strato ein und führe aus:
   ```bash
   mkdir -p ~/.ssh
   chmod 700 ~/.ssh
   ```

4. **Key eintragen:**
   Öffne die Datei auf Strato:
   ```bash
   nano ~/.ssh/authorized_keys
   ```
   *Füge den kopierten Public Key in eine neue Zeile ein, speichere und schließe die Datei.*

5. **Berechtigungen setzen (wichtig!):**
   ```bash
   chmod 600 ~/.ssh/authorized_keys
   ```

Testlauf vom Raspberry Pi aus:
`ssh deine-domain.de@ssh.strato.de`
Du solltest jetzt eingeloggt werden, ohne nach einem Passwort gefragt zu werden.
