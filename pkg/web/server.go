package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"GoSmartMeterGo/pkg/backup"
	"GoSmartMeterGo/pkg/config"
	"GoSmartMeterGo/pkg/database"
	"GoSmartMeterGo/pkg/store"
)

//go:embed assets/*
var assets embed.FS

// Server wraps the HTTP server configuration and handlers.
type Server struct {
	cfg          *config.WebConfig
	store        *store.LiveStore
	db           *database.DBManager
	backupWorker *backup.Worker
}

// NewServer initializes the Web dashboard server.
func NewServer(cfg *config.WebConfig, liveStore *store.LiveStore, db *database.DBManager, backupWorker *backup.Worker) *Server {
	return &Server{
		cfg:          cfg,
		store:        liveStore,
		db:           db,
		backupWorker: backupWorker,
	}
}

// Start launches the HTTP server and blocks until context is cancelled or server errors out.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static Assets
	mux.Handle("/assets/", http.FileServer(http.FS(assets)))

	// Root Index.html redirect
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := assets.ReadFile("assets/index.html")
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// API endpoints
	mux.HandleFunc("/api/live/stream", s.handleLiveStream)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/daily", s.handleDaily)
	mux.HandleFunc("/api/monthly", s.handleMonthly)
	mux.HandleFunc("/api/backup", s.handleBackup)
	mux.HandleFunc("/api/backup/upload", s.handleBackupUpload)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	slog.Info("Web Dashboard server starting", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleLiveStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to store updates for real-time pushing
	updateChan, unsubscribe := s.store.Subscribe()
	defer unsubscribe()

	// Helper to fetch and write the current state
	sendUpdate := func() error {
		sml, sma := s.store.GetLatest()

		smlActive := !sml.Timestamp.IsZero() && time.Since(sml.Timestamp) < 10*time.Second
		smaActive := !sma.Timestamp.IsZero() && time.Since(sma.Timestamp) < 10*time.Second

		data := map[string]any{
			"sml": map[string]any{
				"timestamp":         sml.Timestamp.Format(time.RFC3339),
				"sml_active":        smlActive,
				"sml_power_import":  sml.ActivePowerImport,
				"sml_power_export":  sml.ActivePowerExport,
				"sml_energy_import": sml.ActiveEnergyImport,
				"sml_energy_export": sml.ActiveEnergyExport,
				"sml_power_l1":      sml.PowerL1,
				"sml_power_l2":      sml.PowerL2,
				"sml_power_l3":      sml.PowerL3,
				"sml_voltage_l1":    sml.VoltageL1,
				"sml_voltage_l2":    sml.VoltageL2,
				"sml_voltage_l3":    sml.VoltageL3,
				"sml_current_l1":    sml.CurrentL1,
				"sml_current_l2":    sml.CurrentL2,
				"sml_current_l3":    sml.CurrentL3,
				"sml_frequency":     sml.GridFrequency,
				"sml_interval_ms":   sml.UpdateIntervalMs,
			},
			"sma": map[string]any{
				"timestamp":           sma.Timestamp.Format(time.RFC3339),
				"sma_active":          smaActive,
				"sma_power_import":   sma.ActivePowerImport,
				"sma_power_export":   sma.ActivePowerExport,
				"sma_energy_import":  sma.ActiveEnergyImport,
				"sma_energy_export":  sma.ActiveEnergyExport,
				"sma_reactive_import": sma.ReactivePowerImport,
				"sma_reactive_export": sma.ReactivePowerExport,
				"sma_power_l1":       sma.PowerL1,
				"sma_power_l2":       sma.PowerL2,
				"sma_power_l3":       sma.PowerL3,
				"sma_voltage_l1":     sma.VoltageL1,
				"sma_voltage_l2":     sma.VoltageL2,
				"sma_voltage_l3":     sma.VoltageL3,
				"sma_current_l1":     sma.CurrentL1,
				"sma_current_l2":     sma.CurrentL2,
				"sma_current_l3":     sma.CurrentL3,
				"sma_frequency":      sma.GridFrequency,
				"sma_interval_ms":    sma.UpdateIntervalMs,
			},
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			slog.Error("Failed to marshal live stream data", "error", err)
			return err
		}

		_, err = fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	// Send initial data immediately upon connecting
	if err := sendUpdate(); err != nil {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-updateChan:
			if err := sendUpdate(); err != nil {
				return // Connection probably closed
			}
		}
	}
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	
	// Parse the date parameter in local time, default to today if not provided or invalid
	targetDate, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		now := time.Now()
		targetDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	}

	start := targetDate
	end := targetDate.Add(24*time.Hour - time.Nanosecond)

	history, err := s.db.GetHistory(start, end)
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (s *Server) handleDaily(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var start, end time.Time
	if startStr != "" && endStr != "" {
		var err error
		start, err = time.ParseInLocation("2006-01-02", startStr, time.Local)
		if err == nil {
			end, err = time.ParseInLocation("2006-01-02", endStr, time.Local)
			if err == nil {
				end = end.Add(24*time.Hour - time.Nanosecond)
			}
		}
		if err != nil {
			http.Error(w, "invalid date format", http.StatusBadRequest)
			return
		}
	} else {
		daysStr := r.URL.Query().Get("days")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days <= 0 {
			days = 7
		}
		end = time.Now()
		start = end.AddDate(0, 0, -days)
	}

	daily, err := s.db.GetDailyUsage(start, end)
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(daily)
}

func (s *Server) handleMonthly(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 {
		year = time.Now().Year()
	}

	monthly, err := s.db.GetMonthlyUsage(year)
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monthly)
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	tempFile, err := os.CreateTemp("", "smartmeter-backup-*.db")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp file: %v", err), http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if err := s.db.Backup(tempPath); err != nil {
		http.Error(w, fmt.Sprintf("database backup failed: %v", err), http.StatusInternalServerError)
		return
	}

	file, err := os.Open(tempPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open backup: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=smartmeter-backup-%s.db", time.Now().Format("2006-01-02-15-04-05")))
	w.Header().Set("Expires", "0")
	w.Header().Set("Cache-Control", "must-revalidate")
	w.Header().Set("Pragma", "public")

	if _, err := io.Copy(w, file); err != nil {
		slog.Error("Failed to stream backup to client", "error", err)
	}
}

func (s *Server) handleBackupUpload(w http.ResponseWriter, r *http.Request) {
	if s.backupWorker == nil {
		http.Error(w, "Backup service is not enabled or configured", http.StatusBadRequest)
		return
	}

	slog.Info("Manual backup upload triggered via web API")
	if err := s.backupWorker.Trigger(); err != nil {
		slog.Error("Manual backup upload failed", "error", err)
		http.Error(w, fmt.Sprintf("Backup failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Backup uploaded successfully"})
}
