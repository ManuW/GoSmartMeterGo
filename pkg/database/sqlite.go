package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite"

	"GoSmartMeterGo/pkg/config"
	"GoSmartMeterGo/pkg/store"
)

// MetricsSummary represents the 5-minute aggregated data.
type MetricsSummary struct {
	Timestamp            time.Time `json:"timestamp"`
	SMLImportWh          float64   `json:"sml_import_wh"`
	SMLExportWh          float64   `json:"sml_export_wh"`
	SMAImportWh          float64   `json:"sma_import_wh"`
	SMAExportWh          float64   `json:"sma_export_wh"`
	SMLPowerImportWMin   float64   `json:"sml_power_import_w_min"`
	SMLPowerImportWMax   float64   `json:"sml_power_import_w_max"`
	SMLPowerImportWAvg   float64   `json:"sml_power_import_w_avg"`
	SMLPowerExportWMin   float64   `json:"sml_power_export_w_min"`
	SMLPowerExportWMax   float64   `json:"sml_power_export_w_max"`
	SMLPowerExportWAvg   float64   `json:"sml_power_export_w_avg"`
	SMAPowerImportWMin   float64   `json:"sma_power_import_w_min"`
	SMAPowerImportWMax   float64   `json:"sma_power_import_w_max"`
	SMAPowerImportWAvg   float64   `json:"sma_power_import_w_avg"`
	SMAPowerExportWMin   float64   `json:"sma_power_export_w_min"`
	SMAPowerExportWMax   float64   `json:"sma_power_export_w_max"`
	SMAPowerExportWAvg   float64   `json:"sma_power_export_w_avg"`
}

// DailyUsage represents consumption/production for a single day.
type DailyUsage struct {
	Date           string  `json:"date"`
	SMLConsumedWh  float64 `json:"sml_consumed_wh"`
	SMLDeliveredWh float64 `json:"sml_delivered_wh"`
	SMAConsumedWh  float64 `json:"sma_consumed_wh"`
	SMADeliveredWh float64 `json:"sma_delivered_wh"`
}

// DBManager handles SQLite connection, schema, insertions, and queries.
type DBManager struct {
	db *sql.DB
}

// NewDBManager opens the SQLite database and runs migrations.
func NewDBManager(cfg *config.DatabaseConfig) (*DBManager, error) {
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for concurrency and performance
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		slog.Warn("Failed to enable WAL mode on SQLite", "error", err)
	}

	manager := &DBManager{db: db}
	if err := manager.createSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return manager, nil
}

// Close closes the database connection.
func (m *DBManager) Close() error {
	return m.db.Close()
}

func (m *DBManager) createSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS metrics_summary (
		timestamp DATETIME PRIMARY KEY,
		sml_import_wh REAL,
		sml_export_wh REAL,
		sma_import_wh REAL,
		sma_export_wh REAL,
		sml_power_import_w_min REAL,
		sml_power_import_w_max REAL,
		sml_power_import_w_avg REAL,
		sml_power_export_w_min REAL,
		sml_power_export_w_max REAL,
		sml_power_export_w_avg REAL,
		sma_power_import_w_min REAL,
		sma_power_import_w_max REAL,
		sma_power_import_w_avg REAL,
		sma_power_export_w_min REAL,
		sma_power_export_w_max REAL,
		sma_power_export_w_avg REAL
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics_summary(timestamp);
	`
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create database tables: %w", err)
	}

	// Try to add the columns to existing tables (will error out if they already exist, which we can ignore)
	_, _ = m.db.Exec("ALTER TABLE metrics_summary ADD COLUMN sma_import_wh REAL DEFAULT 0;")
	_, _ = m.db.Exec("ALTER TABLE metrics_summary ADD COLUMN sma_export_wh REAL DEFAULT 0;")

	return nil
}

// InsertSummary saves a MetricsSummary record.
func (m *DBManager) InsertSummary(s *MetricsSummary) error {
	query := `
	INSERT INTO metrics_summary (
		timestamp, sml_import_wh, sml_export_wh, sma_import_wh, sma_export_wh,
		sml_power_import_w_min, sml_power_import_w_max, sml_power_import_w_avg,
		sml_power_export_w_min, sml_power_export_w_max, sml_power_export_w_avg,
		sma_power_import_w_min, sma_power_import_w_max, sma_power_import_w_avg,
		sma_power_export_w_min, sma_power_export_w_max, sma_power_export_w_avg
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(timestamp) DO UPDATE SET
		sml_import_wh=excluded.sml_import_wh,
		sml_export_wh=excluded.sml_export_wh,
		sma_import_wh=excluded.sma_import_wh,
		sma_export_wh=excluded.sma_export_wh,
		sml_power_import_w_min=excluded.sml_power_import_w_min,
		sml_power_import_w_max=excluded.sml_power_import_w_max,
		sml_power_import_w_avg=excluded.sml_power_import_w_avg,
		sml_power_export_w_min=excluded.sml_power_export_w_min,
		sml_power_export_w_max=excluded.sml_power_export_w_max,
		sml_power_export_w_avg=excluded.sml_power_export_w_avg,
		sma_power_import_w_min=excluded.sma_power_import_w_min,
		sma_power_import_w_max=excluded.sma_power_import_w_max,
		sma_power_import_w_avg=excluded.sma_power_import_w_avg,
		sma_power_export_w_min=excluded.sma_power_export_w_min,
		sma_power_export_w_max=excluded.sma_power_export_w_max,
		sma_power_export_w_avg=excluded.sma_power_export_w_avg;
	`
	_, err := m.db.Exec(query,
		s.Timestamp.Format(time.RFC3339), s.SMLImportWh, s.SMLExportWh, s.SMAImportWh, s.SMAExportWh,
		s.SMLPowerImportWMin, s.SMLPowerImportWMax, s.SMLPowerImportWAvg,
		s.SMLPowerExportWMin, s.SMLPowerExportWMax, s.SMLPowerExportWAvg,
		s.SMAPowerImportWMin, s.SMAPowerImportWMax, s.SMAPowerImportWAvg,
		s.SMAPowerExportWMin, s.SMAPowerExportWMax, s.SMAPowerExportWAvg,
	)
	if err != nil {
		return fmt.Errorf("failed to insert summary: %w", err)
	}
	return nil
}

// GetHistory returns aggregated summaries in the time range.
func (m *DBManager) GetHistory(start, end time.Time) ([]MetricsSummary, error) {
	query := `
	SELECT 
		timestamp, sml_import_wh, sml_export_wh, sma_import_wh, sma_export_wh,
		sml_power_import_w_min, sml_power_import_w_max, sml_power_import_w_avg,
		sml_power_export_w_min, sml_power_export_w_max, sml_power_export_w_avg,
		sma_power_import_w_min, sma_power_import_w_max, sma_power_import_w_avg,
		sma_power_export_w_min, sma_power_export_w_max, sma_power_export_w_avg
	FROM metrics_summary
	WHERE timestamp >= ? AND timestamp <= ?
	ORDER BY timestamp ASC
	`
	rows, err := m.db.Query(query, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MetricsSummary
	for rows.Next() {
		var s MetricsSummary
		var tsStr string
		err := rows.Scan(
			&tsStr, &s.SMLImportWh, &s.SMLExportWh, &s.SMAImportWh, &s.SMAExportWh,
			&s.SMLPowerImportWMin, &s.SMLPowerImportWMax, &s.SMLPowerImportWAvg,
			&s.SMLPowerExportWMin, &s.SMLPowerExportWMax, &s.SMLPowerExportWAvg,
			&s.SMAPowerImportWMin, &s.SMAPowerImportWMax, &s.SMAPowerImportWAvg,
			&s.SMAPowerExportWMin, &s.SMAPowerExportWMax, &s.SMAPowerExportWAvg,
		)
		if err != nil {
			return nil, err
		}
		s.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		result = append(result, s)
	}

	return result, nil
}

// GetDailyUsage returns daily summaries for consumed/delivered energy.
func (m *DBManager) GetDailyUsage(start, end time.Time) ([]DailyUsage, error) {
	query := `
	WITH daily_max AS (
		SELECT 
			strftime('%Y-%m-%d', timestamp, 'localtime') as date,
			MAX(sml_import_wh) as max_sml_in,
			MIN(CASE WHEN sml_import_wh > 0 THEN sml_import_wh END) as min_sml_in,
			MAX(sml_export_wh) as max_sml_out,
			MIN(CASE WHEN sml_export_wh > 0 THEN sml_export_wh END) as min_sml_out,
			MAX(sma_import_wh) as max_sma_in,
			MIN(CASE WHEN sma_import_wh > 0 THEN sma_import_wh END) as min_sma_in,
			MAX(sma_export_wh) as max_sma_out,
			MIN(CASE WHEN sma_export_wh > 0 THEN sma_export_wh END) as min_sma_out
		FROM metrics_summary
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY date
	)
	SELECT 
		date,
		COALESCE(CASE 
			WHEN LAG(max_sml_in) OVER (ORDER BY date) > 0 AND max_sml_in >= LAG(max_sml_in) OVER (ORDER BY date)
			THEN max_sml_in - LAG(max_sml_in) OVER (ORDER BY date)
			ELSE max_sml_in - min_sml_in
		END, 0) as sml_consumed,
		COALESCE(CASE 
			WHEN LAG(max_sml_out) OVER (ORDER BY date) > 0 AND max_sml_out >= LAG(max_sml_out) OVER (ORDER BY date)
			THEN max_sml_out - LAG(max_sml_out) OVER (ORDER BY date)
			ELSE max_sml_out - min_sml_out
		END, 0) as sml_delivered,
		COALESCE(CASE 
			WHEN LAG(max_sma_in) OVER (ORDER BY date) > 0 AND max_sma_in >= LAG(max_sma_in) OVER (ORDER BY date)
			THEN max_sma_in - LAG(max_sma_in) OVER (ORDER BY date)
			ELSE max_sma_in - min_sma_in
		END, 0) as sma_consumed,
		COALESCE(CASE 
			WHEN LAG(max_sma_out) OVER (ORDER BY date) > 0 AND max_sma_out >= LAG(max_sma_out) OVER (ORDER BY date)
			THEN max_sma_out - LAG(max_sma_out) OVER (ORDER BY date)
			ELSE max_sma_out - min_sma_out
		END, 0) as sma_delivered
	FROM daily_max
	ORDER BY date ASC
	`
	rows, err := m.db.Query(query, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailyUsage
	for rows.Next() {
		var u DailyUsage
		err := rows.Scan(&u.Date, &u.SMLConsumedWh, &u.SMLDeliveredWh, &u.SMAConsumedWh, &u.SMADeliveredWh)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}

	return result, nil
}

// GetMonthlyUsage returns monthly summaries for consumed/delivered energy for a given year.
func (m *DBManager) GetMonthlyUsage(year int) ([]DailyUsage, error) {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.Local)

	query := `
	WITH monthly_max AS (
		SELECT 
			strftime('%Y-%m', timestamp, 'localtime') as date,
			MAX(sml_import_wh) as max_sml_in,
			MIN(CASE WHEN sml_import_wh > 0 THEN sml_import_wh END) as min_sml_in,
			MAX(sml_export_wh) as max_sml_out,
			MIN(CASE WHEN sml_export_wh > 0 THEN sml_export_wh END) as min_sml_out,
			MAX(sma_import_wh) as max_sma_in,
			MIN(CASE WHEN sma_import_wh > 0 THEN sma_import_wh END) as min_sma_in,
			MAX(sma_export_wh) as max_sma_out,
			MIN(CASE WHEN sma_export_wh > 0 THEN sma_export_wh END) as min_sma_out
		FROM metrics_summary
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY date
	)
	SELECT 
		date,
		COALESCE(CASE 
			WHEN LAG(max_sml_in) OVER (ORDER BY date) > 0 AND max_sml_in >= LAG(max_sml_in) OVER (ORDER BY date)
			THEN max_sml_in - LAG(max_sml_in) OVER (ORDER BY date)
			ELSE max_sml_in - min_sml_in
		END, 0) as sml_consumed,
		COALESCE(CASE 
			WHEN LAG(max_sml_out) OVER (ORDER BY date) > 0 AND max_sml_out >= LAG(max_sml_out) OVER (ORDER BY date)
			THEN max_sml_out - LAG(max_sml_out) OVER (ORDER BY date)
			ELSE max_sml_out - min_sml_out
		END, 0) as sml_delivered,
		COALESCE(CASE 
			WHEN LAG(max_sma_in) OVER (ORDER BY date) > 0 AND max_sma_in >= LAG(max_sma_in) OVER (ORDER BY date)
			THEN max_sma_in - LAG(max_sma_in) OVER (ORDER BY date)
			ELSE max_sma_in - min_sma_in
		END, 0) as sma_consumed,
		COALESCE(CASE 
			WHEN LAG(max_sma_out) OVER (ORDER BY date) > 0 AND max_sma_out >= LAG(max_sma_out) OVER (ORDER BY date)
			THEN max_sma_out - LAG(max_sma_out) OVER (ORDER BY date)
			ELSE max_sma_out - min_sma_out
		END, 0) as sma_delivered
	FROM monthly_max
	ORDER BY date ASC
	`
	rows, err := m.db.Query(query, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailyUsage
	for rows.Next() {
		var u DailyUsage
		err := rows.Scan(&u.Date, &u.SMLConsumedWh, &u.SMLDeliveredWh, &u.SMAConsumedWh, &u.SMADeliveredWh)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}

	return result, nil
}

// Backup creates a safe consistent copy of the database at the target path using VACUUM INTO.
func (m *DBManager) Backup(destPath string) error {
	// Remove destination file if it exists, as VACUUM INTO fails if target exists
	if _, err := os.Stat(destPath); err == nil {
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("failed to remove existing backup file: %w", err)
		}
	}
	_, err := m.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", destPath))
	return err
}

// AggregateAndStore pops the history from store and aggregates it, storing it in SQLite.
func (m *DBManager) AggregateAndStore(liveStore *store.LiveStore) error {
	sml, sma := liveStore.PopHistory()

	if len(sml) == 0 && len(sma) == 0 {
		// Nothing to aggregate
		return nil
	}

	summary := &MetricsSummary{
		Timestamp: time.Now().Truncate(5 * time.Minute),
	}

	// Aggregate SML
	if len(sml) > 0 {
		var sumImport, sumExport float64
		var countImport, countExport float64
		minImport, maxImport := math.MaxFloat64, 0.0
		minExport, maxExport := math.MaxFloat64, 0.0

		// Use the latest cumulative energy value in the batch
		summary.SMLImportWh = sml[len(sml)-1].ActiveEnergyImport
		summary.SMLExportWh = sml[len(sml)-1].ActiveEnergyExport

		for _, r := range sml {
			// Import Power stats
			sumImport += r.ActivePowerImport
			countImport++
			if r.ActivePowerImport < minImport {
				minImport = r.ActivePowerImport
			}
			if r.ActivePowerImport > maxImport {
				maxImport = r.ActivePowerImport
			}

			// Export Power stats
			sumExport += r.ActivePowerExport
			countExport++
			if r.ActivePowerExport < minExport {
				minExport = r.ActivePowerExport
			}
			if r.ActivePowerExport > maxExport {
				maxExport = r.ActivePowerExport
			}
		}

		if minImport == math.MaxFloat64 {
			minImport = 0
		}
		if minExport == math.MaxFloat64 {
			minExport = 0
		}

		summary.SMLPowerImportWMin = minImport
		summary.SMLPowerImportWMax = maxImport
		summary.SMLPowerImportWAvg = sumImport / countImport

		summary.SMLPowerExportWMin = minExport
		summary.SMLPowerExportWMax = maxExport
		summary.SMLPowerExportWAvg = sumExport / countExport
	}

	// Aggregate SMA Speedwire
	if len(sma) > 0 {
		var sumImport, sumExport float64
		var countImport, countExport float64
		minImport, maxImport := math.MaxFloat64, 0.0
		minExport, maxExport := math.MaxFloat64, 0.0

		// Use the latest cumulative energy value in the batch
		summary.SMAImportWh = sma[len(sma)-1].ActiveEnergyImport
		summary.SMAExportWh = sma[len(sma)-1].ActiveEnergyExport

		for _, r := range sma {
			sumImport += r.ActivePowerImport
			countImport++
			if r.ActivePowerImport < minImport {
				minImport = r.ActivePowerImport
			}
			if r.ActivePowerImport > maxImport {
				maxImport = r.ActivePowerImport
			}

			sumExport += r.ActivePowerExport
			countExport++
			if r.ActivePowerExport < minExport {
				minExport = r.ActivePowerExport
			}
			if r.ActivePowerExport > maxExport {
				maxExport = r.ActivePowerExport
			}
		}

		if minImport == math.MaxFloat64 {
			minImport = 0
		}
		if minExport == math.MaxFloat64 {
			minExport = 0
		}

		summary.SMAPowerImportWMin = minImport
		summary.SMAPowerImportWMax = maxImport
		summary.SMAPowerImportWAvg = sumImport / countImport

		summary.SMAPowerExportWMin = minExport
		summary.SMAPowerExportWMax = maxExport
		summary.SMAPowerExportWAvg = sumExport / countExport
	}

	// Save to DB
	if err := m.InsertSummary(summary); err != nil {
		return fmt.Errorf("aggregation save error: %w", err)
	}

	slog.Info("Successfully aggregated and saved 5-minute metrics summary", "timestamp", summary.Timestamp)
	return nil
}
