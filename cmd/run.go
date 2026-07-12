package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"GoSmartMeterGo/pkg/backup"
	"GoSmartMeterGo/pkg/database"
	"GoSmartMeterGo/pkg/sml"
	"GoSmartMeterGo/pkg/speedwire"
	"GoSmartMeterGo/pkg/store"
	"GoSmartMeterGo/pkg/telegram"
	"GoSmartMeterGo/pkg/web"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the smart meter monitoring daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("Starting GoSmartMeterGo daemon...")

		// 1. Initialize thread-safe In-Memory Store
		liveStore := store.NewLiveStore()

		// 2. Initialize Database Manager (SQLite)
		db, err := database.NewDBManager(&Cfg.Database)
		if err != nil {
			return fmt.Errorf("database init failed: %w", err)
		}
		defer db.Close()

		// 3. Initialize Telegram Bot Wrapper
		bot, err := telegram.NewBotWrapper(&Cfg.Telegram)
		if err != nil {
			return fmt.Errorf("telegram bot init failed: %w", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 4. Start SMA Speedwire UDP listener
		speedwireListener := speedwire.NewListener(&Cfg.Speedwire, liveStore)
		go func() {
			if err := speedwireListener.Start(ctx); err != nil {
				slog.Error("Speedwire listener stopped with error", "error", err)
			}
		}()

		// 5. Start SML Serial Reader
		smlReader := sml.NewReader(&Cfg.Serial, liveStore)
		go func() {
			if err := smlReader.Start(ctx); err != nil {
				slog.Error("SML reader stopped with error", "error", err)
			}
		}()

		// 6. Start Aggregation Loop (every 5 minutes)
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					slog.Info("Running periodic aggregation...")
					if err := db.AggregateAndStore(liveStore); err != nil {
						slog.Error("Failed to aggregate and store metrics", "error", err)
					}
				}
			}
		}()

		// 7. Start Telegram Watchdog & Daily Reporter Loop
		if Cfg.Telegram.Enabled {
			go runTelegramServices(ctx, bot, liveStore, db, Cfg.Speedwire.TimeoutSeconds)
		}

		// 7b. Start Backup Worker Loop
		var backupWorker *backup.Worker
		if Cfg.Backup.Enabled {
			backupWorker = backup.NewWorker(&Cfg.Backup, db)
			go backupWorker.Start(ctx)
		}

		// 8. Start Web Server
		webServer := web.NewServer(&Cfg.Web, liveStore, db, backupWorker)
		go func() {
			if err := webServer.Start(); err != nil {
				slog.Error("Web server stopped with error", "error", err)
			}
		}()

		// 9. Signal Handling for Graceful Shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		slog.Info("Daemon is running. Press CTRL+C to stop.")
		sig := <-sigChan
		slog.Info("Received shutdown signal, stopping all services...", "signal", sig)

		cancel()
		
		// Perform final aggregation before exit
		slog.Info("Performing final aggregation...")
		if err := db.AggregateAndStore(liveStore); err != nil {
			slog.Error("Final aggregation failed", "error", err)
		}

		// Allow minor grace period for goroutines to clean up
		time.Sleep(1 * time.Second)
		slog.Info("GoSmartMeterGo stopped gracefully.")
		return nil
	},
}

func runTelegramServices(ctx context.Context, bot *telegram.BotWrapper, liveStore *store.LiveStore, db *database.DBManager, timeoutSecs int) {
	// Start bot updates listener (non-blocking)
	go bot.Start(ctx)

	watchdogTicker := time.NewTicker(15 * time.Second)
	defer watchdogTicker.Stop()

	reportTicker := time.NewTicker(1 * time.Hour)
	defer reportTicker.Stop()

	smaAlertSent := false
	lastReportedDay := time.Now().YearDay()

	for {
		select {
		case <-ctx.Done():
			return
		case <-watchdogTicker.C:
			// SMA Watchdog
			_, speedwire := liveStore.GetLatest()
			if speedwire.Timestamp.IsZero() {
				// No data received yet
				continue
			}

			durationSinceLastPacket := time.Since(speedwire.Timestamp)
			timeout := time.Duration(timeoutSecs) * time.Second

			if durationSinceLastPacket > timeout {
				if !smaAlertSent {
					msg := fmt.Sprintf("⚠️ *Warnung: SMA HomeManager 2.0 offline!*\nSeit %s wurden keine Multicast-Daten mehr empfangen.", durationSinceLastPacket.Round(time.Second))
					bot.SendMessage(ctx, msg)
					smaAlertSent = true
					slog.Warn("SMA watchdog fired: alert sent via Telegram")
				}
			} else {
				if smaAlertSent {
					msg := "✅ *Entwarnung: SMA HomeManager 2.0 online!*\nDer Empfang von Multicast-Daten wurde wiederhergestellt."
					bot.SendMessage(ctx, msg)
					smaAlertSent = false
					slog.Info("SMA watchdog recovered: recovery message sent via Telegram")
				}
			}

		case <-reportTicker.C:
			// Daily Report (send after midnight or at end of day)
			now := time.Now()
			if now.YearDay() != lastReportedDay {
				// Send report for yesterday
				yesterday := now.AddDate(0, 0, -1)
				start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
				end := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location())

				usage, err := db.GetDailyUsage(start, end)
				if err != nil {
					slog.Error("Failed to fetch daily usage for Telegram report", "error", err)
					continue
				}

				if len(usage) > 0 {
					rep := usage[0]
					msg := fmt.Sprintf("📊 *Tagesbericht für den %s:*\n\n*SmartMeter (SML):*\n🔹 *Netzbezug:* %.2f kWh\n🔸 *Einspeisung:* %.2f kWh\n\n*HomeManager (SMA):*\n🔹 *Netzbezug:* %.2f kWh\n🔸 *Einspeisung:* %.2f kWh",
						rep.Date, rep.SMLConsumedWh/1000.0, rep.SMLDeliveredWh/1000.0, rep.SMAConsumedWh/1000.0, rep.SMADeliveredWh/1000.0)
					bot.SendMessage(ctx, msg)
					slog.Info("Sent daily Telegram report", "date", rep.Date)
				}
				lastReportedDay = now.YearDay()
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
}
