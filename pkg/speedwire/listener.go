package speedwire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"GoSmartMeterGo/pkg/config"
	"GoSmartMeterGo/pkg/store"
)

// Listener listens to SMA Speedwire multicast packets and updates the store.
type Listener struct {
	cfg           *config.SpeedwireConfig
	store         *store.LiveStore
	lastTicker    uint32
	hasLastTicker bool
}

// NewListener creates a new Speedwire listener.
func NewListener(cfg *config.SpeedwireConfig, store *store.LiveStore) *Listener {
	return &Listener{
		cfg:   cfg,
		store: store,
	}
}

// Start runs the UDP multicast listener loop until the context is canceled.
func (l *Listener) Start(ctx context.Context) error {
	addrStr := fmt.Sprintf("%s:%d", l.cfg.MulticastGroup, l.cfg.Port)
	// Resolve multicast address
	addr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return err
	}

	buf := make([]byte, 2048)

	// Outer loop for automatic reconnects
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Listen on the multicast group
		conn, err := net.ListenMulticastUDP("udp", nil, addr)
		if err != nil {
			slog.Error("Failed to listen on multicast UDP, retrying in 5s", "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
			continue
		}

		slog.Info("Speedwire listener started", "group", l.cfg.MulticastGroup, "port", l.cfg.Port)

		err = conn.SetReadBuffer(65535)
		if err != nil {
			slog.Warn("Failed to set UDP read buffer size", "error", err)
		}

		// Goroutine to handle context cancellation without leaking
		connDone := make(chan struct{})
		go func(c *net.UDPConn) {
			select {
			case <-ctx.Done():
				c.Close()
			case <-connDone:
				// Listener loop closed the connection and told us to exit
			}
		}(conn)

		// Inner loop for reading packets
		for {
			if ctx.Err() != nil {
				close(connDone)
				conn.Close()
				return ctx.Err()
			}

			// Set a read deadline. Speedwire packets arrive ~every 1 second.
			// If we don't hear anything for 15 seconds, assume the IGMP subscription 
			// dropped (e.g. switch rebooted) and restart the socket.
			conn.SetReadDeadline(time.Now().Add(15 * time.Second))

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				if ctx.Err() != nil {
					close(connDone)
					conn.Close()
					return nil // graceful shutdown
				}

				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					slog.Warn("No Speedwire data received for 15s, reconnecting to refresh IGMP...")
					break // Break inner loop, will close connection and retry
				}

				slog.Error("Error reading UDP multicast packet", "error", err)
				time.Sleep(1 * time.Second) // Prevent tight loop on persistent read error
				continue
			}

			// Parse the packet
			packetData := make([]byte, n)
			copy(packetData, buf[:n])

			packet, err := ParsePacket(packetData)
			if err != nil {
				if errors.Is(err, ErrNotTelemetry) {
					// Silently skip non-telemetry packets (like Discovery or Status Handshakes)
					continue
				}
				slog.Debug("Failed to parse Speedwire packet", "error", err)
				continue
			}

			// Update the store
			var intervalMs uint32
			if l.hasLastTicker {
				// standard 32-bit unsigned wrapping subtraction handles the rollover automatically
				intervalMs = packet.Ticker - l.lastTicker
			}
			l.lastTicker = packet.Ticker
			l.hasLastTicker = true

			reading := store.SpeedwireReading{
				Timestamp:           packet.Timestamp,
				ActivePowerImport:   packet.ActivePowerImport,
				ActivePowerExport:   packet.ActivePowerExport,
				ActiveEnergyImport:  packet.ActiveEnergyImport,
				ActiveEnergyExport:  packet.ActiveEnergyExport,
				ReactivePowerImport: packet.ReactivePowerImport,
				ReactivePowerExport: packet.ReactivePowerExport,
				GridFrequency:       packet.GridFrequency,
				PowerL1:             packet.PowerL1,
				PowerL2:             packet.PowerL2,
				PowerL3:             packet.PowerL3,
				VoltageL1:           packet.VoltageL1,
				VoltageL2:           packet.VoltageL2,
				VoltageL3:           packet.VoltageL3,
				CurrentL1:           packet.CurrentL1,
				CurrentL2:           packet.CurrentL2,
				CurrentL3:           packet.CurrentL3,
				UpdateIntervalMs:    intervalMs,
			}
			l.store.UpdateSpeedwire(reading)

			slog.Debug("Speedwire data received",
				"serial", packet.SerialNumber,
				"importW", packet.ActivePowerImport,
				"exportW", packet.ActivePowerExport,
			)
		}

		// Close connection and signal goroutine to exit before restarting the outer loop
		close(connDone)
		conn.Close()
	}
}
