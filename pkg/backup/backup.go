package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"GoSmartMeterGo/pkg/config"
	"GoSmartMeterGo/pkg/database"
)

type Worker struct {
	cfg *config.BackupConfig
	db  *database.DBManager
}

func NewWorker(cfg *config.BackupConfig, db *database.DBManager) *Worker {
	return &Worker{
		cfg: cfg,
		db:  db,
	}
}

func (w *Worker) Start(ctx context.Context) {
	slog.Info("Backup worker started", "interval_hours", w.cfg.IntervalHours, "provider", w.cfg.Provider)

	// Starte das erste Backup nach 10 Sekunden Verzögerung
	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second):
		_ = w.runBackup()
	}

	ticker := time.NewTicker(time.Duration(w.cfg.IntervalHours) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Backup worker stopping...")
			return
		case <-ticker.C:
			_ = w.runBackup()
		}
	}
}

// Trigger triggers a database backup and sync run manually, returning any error.
func (w *Worker) Trigger() error {
	return w.runBackup()
}

func (w *Worker) runBackup() error {
	slog.Info("Starting database backup process...")
	
	tempDbPath := filepath.Join(os.TempDir(), "smartmeter_sync.db")
	
	if err := w.db.Backup(tempDbPath); err != nil {
		slog.Error("Database vacuum failed", "error", err)
		return fmt.Errorf("database vacuum failed: %w", err)
	}
	defer os.Remove(tempDbPath)
	
	slog.Info("Database vacuum complete, temp file created", "path", tempDbPath)

	if strings.ToLower(w.cfg.Provider) == "rsync" {
		if err := w.runRsync(tempDbPath); err != nil {
			slog.Error("Rsync backup failed", "error", err)
			return err
		}
		slog.Info("Database backup successfully synchronized to remote host.")
	} else {
		slog.Error("Unsupported backup provider configured", "provider", w.cfg.Provider)
		return fmt.Errorf("unsupported provider: %s", w.cfg.Provider)
	}
	return nil
}

func (w *Worker) runRsync(localPath string) error {
	rCfg := w.cfg.Rsync
	if rCfg.Host == "" || rCfg.Username == "" || rCfg.RemotePath == "" {
		return fmt.Errorf("rsync is enabled but config is incomplete (host/username/remote_path is empty)")
	}

	// 1. Lese den Namen des letzten erfolgreichen Backups für --link-dest
	statePath := "last_backup.txt"
	lastBackupDirName := ""
	if data, err := os.ReadFile(statePath); err == nil {
		lastBackupDirName = strings.TrimSpace(string(data))
	}

	// 2. Erstelle aktuellen Verzeichnisnamen
	timestamp := time.Now().Format("2006-01-02_15-04")
	newBackupDirName := fmt.Sprintf("smartmeter_%s", timestamp)

	// 2b. Erstelle das Zielverzeichnis auf dem Strato-Server vorab via SSH
	// (rsync legt Verzeichnisse für Dateiziele standardmäßig nicht an)
	mkdirArgs := []string{"-o", "StrictHostKeyChecking=accept-new"}
	if rCfg.SSHKeyPath != "" {
		mkdirArgs = append(mkdirArgs, "-i", rCfg.SSHKeyPath)
	}
	remoteDir := fmt.Sprintf("%s/%s", rCfg.RemotePath, newBackupDirName)
	mkdirArgs = append(mkdirArgs, fmt.Sprintf("%s@%s", rCfg.Username, rCfg.Host), fmt.Sprintf("mkdir -p %s", remoteDir))

	slog.Info("Creating remote backup directory via SSH...", "dir", remoteDir)
	mkdirCmd := exec.Command("ssh", mkdirArgs...)
	if output, err := mkdirCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create remote directory via SSH: %w (output: %s)", err, string(output))
	}

	// 3. rsync Argumente aufbauen
	args := []string{"-avz"}
	
	if lastBackupDirName != "" {
		args = append(args, fmt.Sprintf("--link-dest=../%s", lastBackupDirName))
	}

	sshCmd := "ssh -o StrictHostKeyChecking=accept-new"
	if rCfg.SSHKeyPath != "" {
		sshCmd = fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=accept-new", rCfg.SSHKeyPath)
	}
	args = append(args, "-e", sshCmd)

	remoteDest := fmt.Sprintf("%s@%s:%s/%s/smartmeter_sync.db", rCfg.Username, rCfg.Host, rCfg.RemotePath, newBackupDirName)
	args = append(args, localPath, remoteDest)

	slog.Info("Executing rsync command...", "args", strings.Join(args, " "))
	cmd := exec.Command("rsync", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync execution failed: %w (output: %s)", err, string(output))
	}

	// 4. Speicher aktuellen Ordnernamen für den nächsten Durchlauf
	if err := os.WriteFile(statePath, []byte(newBackupDirName), 0644); err != nil {
		slog.Warn("Failed to write last backup state file", "error", err)
	}

	// 5. Bereinigung alter Backups (Retention)
	if w.cfg.KeepRevisions > 0 {
		w.cleanupOldBackups()
	}

	return nil
}

func (w *Worker) cleanupOldBackups() {
	rCfg := w.cfg.Rsync
	
	if rCfg.RemotePath == "" || rCfg.RemotePath == "/" {
		slog.Error("Aborting remote cleanup: remote_path is unsafe", "path", rCfg.RemotePath)
		return
	}

	// SSH-Befehl zum Löschen überzähliger Backups
	remoteCmd := fmt.Sprintf("dirs=$(ls -1d %s/smartmeter_* 2>/dev/null | sort | head -n -%d); [ -n \"$dirs\" ] && echo \"$dirs\" | xargs rm -rf", rCfg.RemotePath, w.cfg.KeepRevisions)
	
	sshArgs := []string{"-o", "StrictHostKeyChecking=accept-new"}
	if rCfg.SSHKeyPath != "" {
		sshArgs = append(sshArgs, "-i", rCfg.SSHKeyPath)
	}
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", rCfg.Username, rCfg.Host), remoteCmd)

	slog.Info("Executing remote SSH cleanup...", "args", strings.Join(sshArgs, " "))
	cmd := exec.Command("ssh", sshArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Error("Remote cleanup failed", "error", err, "output", string(output))
	} else {
		slog.Info("Remote backup retention cleanup finished successfully.")
	}
}
