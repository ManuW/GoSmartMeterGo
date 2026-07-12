package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"

	"3e8.eu/go/sml"
	"3e8.eu/go/sml/types"
	"go.bug.st/serial"
	"golang.org/x/term"
)

// Global state for the dashboard
var (
	latestValues = make(map[string]valueEntry)
	meterID      string
	uptime       string
	stateMu      sync.Mutex
)

type valueEntry struct {
	Name  string
	Obis  string
	Value string
	Unit  string
}

func main() {
	var device string
	var rawFile string
	var logFile string
	var vtView bool
	var listPorts bool

	flag.StringVar(&device, "port", "/dev/ttyAMA0", "Serial port device (e.g. /dev/ttyAMA0 or - for stdin)")
	flag.StringVar(&rawFile, "raw-file", "", "Optional: File to save raw binary data")
	flag.StringVar(&logFile, "log-file", "", "Optional: File to save human-readable log")
	flag.BoolVar(&vtView, "vt", false, "Enable VT100 dashboard view")
	flag.BoolVar(&listPorts, "list", false, "List available serial ports")
	flag.Parse()

	if listPorts {
		ports, err := serial.GetPortsList()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing ports: %v\n", err)
			os.Exit(1)
		}
		if len(ports) == 0 {
			fmt.Println("No serial ports found.")
		} else {
			fmt.Println("Available serial ports:")
			for _, port := range ports {
				fmt.Printf("  %s\n", port)
			}
		}
		return
	}

	port, err := serialPortOpen(device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer port.Close()

	var rawWriter io.Writer
	if rawFile != "" {
		f, err := os.OpenFile(rawFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening raw file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		rawWriter = f
	}

	var logFileWriter io.Writer
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		logFileWriter = f
	}

	// Setup terminal for interactivity if not reading from stdin
	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) && device != "-" {
		oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool)
	go func() {
		if oldState != nil {
			b := make([]byte, 1)
			for {
				n, _ := os.Stdin.Read(b)
				// Handle 'q', 'Q', and Ctrl+C (\x03)
				if n > 0 && (b[0] == 'q' || b[0] == 'Q' || b[0] == 0x03) {
					break
				}
			}
		} else {
			<-sigChan
		}
		if oldState != nil {
			term.Restore(int(os.Stdin.Fd()), oldState)
		}
		fmt.Print("\033[?25h") // Show cursor
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	if vtView {
		fmt.Print("\033[?25l") // Hide cursor
		fmt.Print("\033[2J")   // Clear screen
	} else {
		fmt.Printf("Listening on %s...\n", device)
	}

	// Wrap reader in TeeReader if we want to save raw binary data
	var reader io.Reader = port
	if rawWriter != nil {
		reader = io.TeeReader(port, rawWriter)
	}

	for {
		data, err := sml.ReadFile(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			if vtView {
				// Don't spam dashboard
			} else {
				fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			}
			continue
		}

		values, err := sml.ParseValues(data)
		if err != nil {
			continue
		}

		messages, err := sml.DecodeMessages(values)
		if err != nil {
			continue
		}

		stateMu.Lock()
		for _, msg := range messages {
			updateState(msg)
		}

		if logFileWriter != nil {
			writeToLog(logFileWriter)
		}

		if vtView {
			renderDashboard()
		} else {
			printLatest(os.Stdout)
		}
		stateMu.Unlock()
	}
	done <- true
}

func serialPortOpen(device string) (io.ReadWriteCloser, error) {
	if device == "-" {
		return os.Stdin, nil
	}
	mode := &serial.Mode{
		BaudRate: 9600,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(device, mode)
	if err != nil {
		return nil, err
	}
	_ = port.SetRTS(true)
	return port, nil
}

func updateState(msg types.SML_Message) {
	if getListRes, ok := msg.MessageBody.Val.(types.SML_GetList_Res); ok {
		// Update Meter ID (Server ID)
		meterID = formatOctetString(getListRes.ServerID)

		// Update Uptime (Sensor Time)
		if getListRes.ActSensorTime.Valid {
			uptime = fmt.Sprintf("%v s", getListRes.ActSensorTime.Val.Val)
		}

		for _, entry := range getListRes.ValList {
			obisHex := fmt.Sprintf("% x", []byte(entry.ObjName))
			name := getObisName(obisHex)
			val := entry.Value.Val
			displayValue := formatValue(val, entry.Scaler)
			unit := getUnitName(entry.Unit)

			if name == "" {
				name = "Unknown"
			}

			latestValues[obisHex] = valueEntry{
				Name:  name,
				Obis:  obisHex,
				Value: displayValue,
				Unit:  unit,
			}
		}
	}
}

func formatOctetString(os types.OctetString) string {
	if isASCII(os) {
		return string(os)
	}
	return fmt.Sprintf("%x", []byte(os))
}

func getUnitName(u types.Optional[types.SML_Unit]) string {
	if !u.Valid {
		return ""
	}
	switch int(u.Val) {
	case 27:
		return "W"
	case 30:
		return "Wh"
	case 33:
		return "A"
	case 35:
		return "V"
	case 44:
		return "Hz"
	case 8:
		return "°"
	default:
		return fmt.Sprintf("(unit %d)", u.Val)
	}
}

func renderDashboard() {
	fmt.Print("\033[H") // Move to 0,0
	fmt.Print("=== SML Smart Meter Dashboard (Press 'q' or Ctrl+C to quit) ===\r\n")
	fmt.Printf("Meter ID: %-20s | Uptime: %s\033[K\r\n", meterID, uptime)
	fmt.Printf("%-25s | %-20s | %s\r\n", "Description", "OBIS", "Value")
	fmt.Printf("%s\r\n", strings.Repeat("-", 65))

	keys := make([]string, 0, len(latestValues))
	for k := range latestValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := latestValues[k]
		fmt.Printf("%-25s | %-20s | %s %s\033[K\r\n", v.Name, v.Obis, v.Value, v.Unit)
	}
	fmt.Print("\033[J") // Clear remaining screen
}

func printLatest(w io.Writer) {
	newline := "\n"
	if w == os.Stdout {
		newline = "\r\n"
	}
	fmt.Fprintf(w, "--- New SML Document ---%s", newline)
	fmt.Fprintf(w, "Meter ID: %s | Uptime: %s%s", meterID, uptime, newline)
	keys := make([]string, 0, len(latestValues))
	for k := range latestValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := latestValues[k]
		fmt.Fprintf(w, "%-25s (%s): %s %s%s", v.Name, v.Obis, v.Value, v.Unit, newline)
	}
}

func writeToLog(w io.Writer) {
	fmt.Fprintln(w, "--- SML Update ---")
	fmt.Fprintf(w, "Meter ID: %s | Uptime: %s\n", meterID, uptime)
	keys := make([]string, 0, len(latestValues))
	for k := range latestValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := latestValues[k]
		fmt.Fprintf(w, "%-25s (%s): %s %s", v.Name, v.Obis, v.Value, v.Unit)
		fmt.Fprintln(w)
	}
}

func formatValue(val interface{}, scaler types.Optional[types.Integer8]) string {
	if val == nil {
		return "nil"
	}
	if b, ok := val.(types.OctetString); ok {
		if isASCII(b) {
			return fmt.Sprintf("\"%s\"", string(b))
		}
		return fmt.Sprintf("%v", []byte(b))
	}
	if scaler.Valid {
		s := int(scaler.Val)
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
			return fmt.Sprintf("%v", val)
		}
		if s != 0 {
			for i := 0; i < s; i++ {
				f *= 10
			}
			for i := 0; i > s; i-- {
				f /= 10
			}
			return fmt.Sprintf("%.3f", f)
		}
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%v", val)
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c < 32 || c > 126 {
			return false
		}
	}
	return len(b) > 0
}

func getObisName(obis string) string {
	mapping := map[string]string{
		"01 00 60 32 01 01": "Herstellerkennung",
		"01 00 60 01 00 ff": "Geräteidentifikation",
		"01 00 01 08 00 ff": "Zählerstand +A",
		"01 00 01 08 01 ff": "Zählerstand +A T1",
		"01 00 01 08 02 ff": "Zählerstand +A T2",
		"01 00 02 08 00 ff": "Zählerstand -A",
		"01 00 00 02 00 00": "Firmware Version",
		"01 00 60 5a 02 01": "Checksumme",
		"01 00 10 07 00 ff": "Wirkleistung gesamt",
		"01 00 24 07 00 ff": "Wirkleistung L1",
		"01 00 38 07 00 ff": "Wirkleistung L2",
		"01 00 4c 07 00 ff": "Wirkleistung L3",
		"01 00 20 07 00 ff": "Spannung L1",
		"01 00 34 07 00 ff": "Spannung L2",
		"01 00 48 07 00 ff": "Spannung L3",
		"01 00 1f 07 00 ff": "Strom L1",
		"01 00 33 07 00 ff": "Strom L2",
		"01 00 47 07 00 ff": "Strom L3",
		"01 00 51 07 01 ff": "Phasenwinkel U-L2/U-L1",
		"01 00 51 07 02 ff": "Phasenwinkel U-L3/U-L1",
		"01 00 51 07 04 ff": "Phasenwinkel I-L1/U-L1",
		"01 00 51 07 0f ff": "Phasenwinkel I-L2/U-L2",
		"01 00 51 07 1a ff": "Phasenwinkel I-L3/U-L3",
		"01 00 0e 07 00 ff": "Frequenz",
	}
	return mapping[obis]
}
