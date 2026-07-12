package speedwire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
)

// Packet represents the parsed metrics from an SMA Speedwire UDP multicast telegram.
type Packet struct {
	Timestamp          time.Time
	SusyID             uint16
	SerialNumber       uint32
	Ticker             uint32
	ActivePowerImport  float64 // W
	ActivePowerExport  float64 // W
	ActiveEnergyImport float64 // Wh
	ActiveEnergyExport float64 // Wh
	ReactivePowerImport float64 // var
	ReactivePowerExport float64 // var
	GridFrequency      float64 // Hz
	PowerL1            float64 // W (signed: Import - Export)
	PowerL2            float64 // W (signed: Import - Export)
	PowerL3            float64 // W (signed: Import - Export)
	VoltageL1          float64 // V
	VoltageL2          float64 // V
	VoltageL3          float64 // V
	CurrentL1          float64 // A
	CurrentL2          float64 // A
	CurrentL3          float64 // A
}

// ErrNotTelemetry indicates that the parsed packet does not contain SMA Net 2 telemetry (SubTag 0x6069).
var ErrNotTelemetry = errors.New("packet does not contain SMA Net 2 telemetry (0x6069)")

// ParsePacket decodes raw Speedwire UDP payload bytes.
func ParsePacket(data []byte) (*Packet, error) {
	if len(data) < 8 {
		return nil, errors.New("packet too short")
	}

	// Verify Start Sequence "SMA\0"
	if !bytes.Equal(data[0:4], []byte("SMA\x00")) {
		return nil, errors.New("invalid start sequence")
	}

	packet := &Packet{
		Timestamp: time.Now(),
	}

	foundTelemetry := false
	idx := 4
	for idx < len(data) {
		if idx+4 > len(data) {
			break
		}

		length := int(binary.BigEndian.Uint16(data[idx : idx+2]))
		tag := binary.BigEndian.Uint16(data[idx+2 : idx+4])
		idx += 4

		if idx+length > len(data) {
			return nil, errors.New("tag length out of bounds")
		}

		payload := data[idx : idx+length]
		idx += length

		// TAG: SMA Net 2
		if tag == 0x0010 {
			if len(payload) < 18 {
				return nil, errors.New("SMA Net 2 payload too short")
			}

			subTag := binary.BigEndian.Uint16(payload[0:2])
			if subTag == 0x6069 {
				packet.SusyID = binary.BigEndian.Uint16(payload[2:4])
				packet.SerialNumber = binary.BigEndian.Uint32(payload[4:8])
				packet.Ticker = binary.BigEndian.Uint32(payload[8:12])
				foundTelemetry = true

				obisData := payload[12:]
				oIdx := 0
				for oIdx < len(obisData) {
					if oIdx+4 > len(obisData) {
						break
					}

					channel := obisData[oIdx]
					index := obisData[oIdx+1]
					valType := obisData[oIdx+2]
					_ = obisData[oIdx+3] // tariff is unused
					oIdx += 4

					valLen := 4
					if valType == 8 {
						valLen = 8
					}

					if oIdx+valLen > len(obisData) {
						break
					}

					valBytes := obisData[oIdx : oIdx+valLen]
					oIdx += valLen

					// Decode values based on OBIS channels and indices
					if channel == 0 {
						switch valType {
						case 4:
							val := float64(binary.BigEndian.Uint32(valBytes))
							switch index {
							case 1: // Total: Active Power Import (+P)
								packet.ActivePowerImport = val / 10.0
							case 2: // Total: Active Power Export (-P)
								packet.ActivePowerExport = val / 10.0
							case 3: // Total: Reactive Power Import (+Q)
								packet.ReactivePowerImport = val / 10.0
							case 4: // Total: Reactive Power Export (-Q)
								packet.ReactivePowerExport = val / 10.0
							case 14: // Grid Frequency
								packet.GridFrequency = val / 1000.0
							case 21: // L1: Active Power Import (+P)
								packet.PowerL1 += val / 10.0
							case 22: // L1: Active Power Export (-P)
								packet.PowerL1 -= val / 10.0
							case 31: // L1: Current
								packet.CurrentL1 = val / 1000.0
							case 32: // L1: Voltage
								packet.VoltageL1 = val / 1000.0
							case 41: // L2: Active Power Import (+P)
								packet.PowerL2 += val / 10.0
							case 42: // L2: Active Power Export (-P)
								packet.PowerL2 -= val / 10.0
							case 51: // L2: Current
								packet.CurrentL2 = val / 1000.0
							case 52: // L2: Voltage
								packet.VoltageL2 = val / 1000.0
							case 61: // L3: Active Power Import (+P)
								packet.PowerL3 += val / 10.0
							case 62: // L3: Active Power Export (-P)
								packet.PowerL3 -= val / 10.0
							case 71: // L3: Current
								packet.CurrentL3 = val / 1000.0
							case 72: // L3: Voltage
								packet.VoltageL3 = val / 1000.0
							}
						case 8:
							val := float64(binary.BigEndian.Uint64(valBytes))
							switch index {
							case 1: // Total: Active Energy Import (Ws)
								packet.ActiveEnergyImport = val / 3600.0 // Convert to Wh
							case 2: // Total: Active Energy Export (Ws)
								packet.ActiveEnergyExport = val / 3600.0 // Convert to Wh
							}
						}
					}
				}
			}
		}
	}

	if !foundTelemetry {
		return nil, ErrNotTelemetry
	}

	return packet, nil
}
