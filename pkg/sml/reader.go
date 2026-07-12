package sml

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	smlparser "3e8.eu/go/sml"
	"3e8.eu/go/sml/types"
	"go.bug.st/serial"

	"GoSmartMeterGo/pkg/config"
	"GoSmartMeterGo/pkg/store"
)

// Reader reads from the serial optical probe and parses SML frames.
type Reader struct {
	cfg      *config.SerialConfig
	store    *store.LiveStore
	lastTime time.Time
}

// NewReader creates a new SML serial reader.
func NewReader(cfg *config.SerialConfig, store *store.LiveStore) *Reader {
	return &Reader{
		cfg:   cfg,
		store: store,
	}
}

// Start opens the serial port and reads SML messages in a loop until the context is canceled.
func (r *Reader) Start(ctx context.Context) error {
	slog.Info("SML Serial Reader starting", "port", r.cfg.Port)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := r.readLoop(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			slog.Error("SML read loop error, retrying in 5 seconds...", "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (r *Reader) readLoop(ctx context.Context) error {
	var stopBits serial.StopBits
	switch r.cfg.StopBits {
	case 1:
		stopBits = serial.OneStopBit
	case 2:
		stopBits = serial.TwoStopBits
	case 15: // 1.5 stop bits
		stopBits = serial.OnePointFiveStopBits
	default:
		slog.Warn("Unsupported stop bits configuration, defaulting to 1 stop bit", "stop_bits", r.cfg.StopBits)
		stopBits = serial.OneStopBit
	}

	mode := &serial.Mode{
		BaudRate: r.cfg.BaudRate,
		DataBits: r.cfg.DataBits,
		StopBits: stopBits,
	}

	switch r.cfg.Parity {
	case "none":
		mode.Parity = serial.NoParity
	case "odd":
		mode.Parity = serial.OddParity
	case "even":
		mode.Parity = serial.EvenParity
	default:
		mode.Parity = serial.NoParity
	}

	port, err := serial.Open(r.cfg.Port, mode)
	if err != nil {
		return err
	}
	defer port.Close()

	if r.cfg.RTS {
		// Set RTS high to power the optical reader probe
		if err := port.SetRTS(true); err != nil {
			slog.Warn("Failed to set RTS on serial port", "error", err)
		}
	}

	slog.Info("SML Serial Port opened successfully", "port", r.cfg.Port)

	// Flush any stale data that accumulated in the OS buffer while the daemon was offline
	if err := port.ResetInputBuffer(); err != nil {
		slog.Warn("Failed to reset serial input buffer", "error", err)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// ReadFile blocks until an SML frame start/end sequence is found
		rawBytes, err := smlparser.ReadFile(port)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return errors.New("serial port EOF")
			}
			return err
		}

		// Parse the raw bytes
		values, err := smlparser.ParseValues(rawBytes)
		if err != nil {
			slog.Debug("SML parse values error", "error", err)
			continue
		}

		messages, err := smlparser.DecodeMessages(values)
		if err != nil {
			slog.Debug("SML decode messages error", "error", err)
			continue
		}

		// Extract metrics manually via OBIS codes as in doc/SML.go
		now := time.Now()
		reading := store.SMLReading{
			Timestamp: now,
		}
		if !r.lastTime.IsZero() {
			reading.UpdateIntervalMs = uint32(now.Sub(r.lastTime).Milliseconds())
		}
		r.lastTime = now

		foundAny := false

		for _, msg := range messages {
			if getListRes, ok := msg.MessageBody.Val.(types.SML_GetList_Res); ok {
				for _, entry := range getListRes.ValList {
					obisHex := fmt.Sprintf("% x", []byte(entry.ObjName))
					val, ok := getFloatValue(entry.Value.Val, entry.Scaler)
					if !ok {
						continue
					}
					foundAny = true

					switch obisHex {
					case "01 00 10 07 00 ff": // Wirkleistung gesamt (+/- W)
						if val >= 0 {
							reading.ActivePowerImport = val
							reading.ActivePowerExport = 0
						} else {
							reading.ActivePowerImport = 0
							reading.ActivePowerExport = -val
						}
					case "01 00 24 07 00 ff": // Wirkleistung L1
						reading.PowerL1 = val
					case "01 00 38 07 00 ff": // Wirkleistung L2
						reading.PowerL2 = val
					case "01 00 4c 07 00 ff": // Wirkleistung L3
						reading.PowerL3 = val
					case "01 00 01 08 00 ff": // Zählerstand +A (Wh)
						reading.ActiveEnergyImport = val
					case "01 00 02 08 00 ff": // Zählerstand -A (Wh)
						reading.ActiveEnergyExport = val
					case "01 00 20 07 00 ff": // Spannung L1 (V)
						reading.VoltageL1 = val
					case "01 00 34 07 00 ff": // Spannung L2 (V)
						reading.VoltageL2 = val
					case "01 00 48 07 00 ff": // Spannung L3 (V)
						reading.VoltageL3 = val
					case "01 00 1f 07 00 ff": // Strom L1 (A)
						reading.CurrentL1 = val
					case "01 00 33 07 00 ff": // Strom L2 (A)
						reading.CurrentL2 = val
					case "01 00 47 07 00 ff": // Strom L3 (A)
						reading.CurrentL3 = val
					case "01 00 0e 07 00 ff": // Frequenz (Hz)
						reading.GridFrequency = val
					}
				}
			}
		}

		if !foundAny {
			continue
		}

		r.store.UpdateSML(reading)

		slog.Debug("SML data parsed and stored",
			"importW", reading.ActivePowerImport,
			"exportW", reading.ActivePowerExport,
			"energyImportWh", reading.ActiveEnergyImport,
			"energyExportWh", reading.ActiveEnergyExport,
		)
	}
}

func getFloatValue(val interface{}, scaler types.Optional[types.Integer8]) (float64, bool) {
	if val == nil {
		return 0, false
	}
	var f float64
	switch v := val.(type) {
	case int64:
		f = float64(v)
	case uint64:
		f = float64(v)
	case int:
		f = float64(v)
	case int32:
		f = float64(v)
	case uint32:
		f = float64(v)
	case types.Integer64:
		f = float64(v)
	case types.Integer32:
		f = float64(v)
	case types.Integer16:
		f = float64(v)
	case types.Integer8:
		f = float64(v)
	case types.Unsigned64:
		f = float64(v)
	case types.Unsigned32:
		f = float64(v)
	case types.Unsigned16:
		f = float64(v)
	case types.Unsigned8:
		f = float64(v)
	default:
		return 0, false
	}

	if scaler.Valid {
		s := int(scaler.Val)
		if s > 0 {
			for i := 0; i < s; i++ {
				f *= 10
			}
		} else if s < 0 {
			for i := 0; i > s; i-- {
				f /= 10
			}
		}
	}
	return f, true
}
