# Smart Message Language (SML) Documentation

This document describes how to interact with and parse data from a Logarex LK13BE smart meter using the SML protocol.

## 1. Serial Interface Configuration

The Logarex LK13BE optical interface typically requires the following serial parameters:

- **Baud Rate**: 9600
- **Data Bits**: 8
- **Parity**: None (N)
- **Stop Bits**: 1
- **Flow Control**: None (Hardware RTS may be required to power the optical probe)

### Implementation in Go
Using the `go.bug.st/serial` library:

```go
mode := &serial.Mode{
    BaudRate: 9600,
    DataBits: 8,
    Parity:   serial.NoParity,
    StopBits: serial.OneStopBit,
}
port, err := serial.Open("/dev/ttyAMA0", mode)
// Often necessary to power the optical reader
port.SetRTS(true)
```

## 2. SML Message Structure

SML is a binary, TLV (Tag-Length-Value) based protocol. A typical SML transmission consists of one or more SML files/documents.

### Escape Sequences
SML messages are framed by a specific escape sequence:
- **Start**: `1b 1b 1b 1b 01 01 01 01`
- **End**: `1b 1b 1b 1b 1a [3 bytes CRC] 00`

### Message Content
A document usually contains:
1. **SML_PublicOpen.Res**: Metadata about the transmission (e.g., Uptime/SensorTime).
2. **SML_GetList.Res**: The actual payload containing meter readings.
3. **SML_PublicClose.Res**: End of document marker.

## 3. Parsing SML Messages

The `3e8.eu/go/sml` library is used to handle the heavy lifting of decoding the binary TLV structures into Go types.

### Decoding Workflow
1. **Extraction**: `sml.ReadFile(reader)` identifies the start/end sequences and returns the raw SML document bytes.
2. **TLV Parsing**: `sml.ParseValues(data)` converts raw bytes into a tree of TLV values.
3. **Type Decoding**: `sml.DecodeMessages(values)` maps TLV values into structured SML message objects.

### Mapping OBIS Codes
OBIS (Object Identification System) codes identify the specific data points. Common codes for the Logarex LK13BE include:

| OBIS Code | Description | Unit |
| :--- | :--- | :--- |
| `01 00 01 08 00 ff` | Total Energy Consumption (+A) | Wh |
| `01 00 02 08 00 ff` | Total Energy Delivery (-A) | Wh |
| `01 00 10 07 00 ff` | Total Active Power | W |
| `01 00 20 07 00 ff` | Voltage L1 | V |
| `01 00 1f 07 00 ff` | Current L1 | A |

### Scaling and Units
SML values are often sent as integers and require a **Scaler** (power of 10) to reach the correct value.

**Example**:
- Raw Value: `2318`
- Scaler: `-1`
- Calculation: `2318 * 10^-1 = 231.8 V`

## 4. Troubleshooting

- **No Data**: Check physical alignment of the optical probe and ensure RTS is enabled to provide power.
- **Staircase Output**: If running in raw terminal mode, ensure your print statements use `\r\n` instead of just `\n`.
- **Permission Denied**: On Linux (Raspberry Pi), ensure the user is part of the `dialout` group to access `/dev/ttyAMA0`.
