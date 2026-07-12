package store

import (
	"sync"
	"time"
)

// SMLReading represents a parsed measurement from the SML optical interface.
type SMLReading struct {
	Timestamp          time.Time
	ActivePowerImport  float64 // W
	ActivePowerExport  float64 // W
	ActiveEnergyImport float64 // Wh
	ActiveEnergyExport float64 // Wh
	PowerL1            float64 // W (signed: Import - Export)
	PowerL2            float64 // W (signed: Import - Export)
	PowerL3            float64 // W (signed: Import - Export)
	VoltageL1          float64 // V
	VoltageL2          float64 // V
	VoltageL3          float64 // V
	CurrentL1          float64 // A
	CurrentL2          float64 // A
	CurrentL3          float64 // A
	GridFrequency      float64 // Hz
	UpdateIntervalMs   uint32  // ms
}

// SpeedwireReading represents a parsed measurement from the SMA Speedwire protocol.
type SpeedwireReading struct {
	Timestamp           time.Time
	ActivePowerImport   float64 // W
	ActivePowerExport   float64 // W
	ActiveEnergyImport  float64 // Wh
	ActiveEnergyExport  float64 // Wh
	ReactivePowerImport float64 // var
	ReactivePowerExport float64 // var
	GridFrequency       float64 // Hz
	PowerL1             float64 // W (signed: Import - Export)
	PowerL2             float64 // W (signed: Import - Export)
	PowerL3             float64 // W (signed: Import - Export)
	VoltageL1           float64 // V
	VoltageL2           float64 // V
	VoltageL3           float64 // V
	CurrentL1           float64 // A
	CurrentL2           float64 // A
	CurrentL3           float64 // A
	UpdateIntervalMs    uint32  // ms
}

// LiveStore maintains the thread-safe state of the latest readings and historical buffers, and handles live subscriptions.
type LiveStore struct {
	mu sync.RWMutex

	latestSML       SMLReading
	latestSpeedwire SpeedwireReading

	smlHistory       []SMLReading
	speedwireHistory []SpeedwireReading

	subscribers []chan struct{}
}

// NewLiveStore initializes a new LiveStore.
func NewLiveStore() *LiveStore {
	return &LiveStore{
		smlHistory:       make([]SMLReading, 0, 300),
		speedwireHistory: make([]SpeedwireReading, 0, 300),
	}
}

// UpdateSML updates the latest SML reading, appends it to the history, and notifies subscribers.
func (s *LiveStore) UpdateSML(r SMLReading) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latestSML = r
	s.smlHistory = append(s.smlHistory, r)
	s.notifySubscribers()
}

// UpdateSpeedwire updates the latest Speedwire reading, appends it to the history, and notifies subscribers.
func (s *LiveStore) UpdateSpeedwire(r SpeedwireReading) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latestSpeedwire = r
	s.speedwireHistory = append(s.speedwireHistory, r)
	s.notifySubscribers()
}

// GetLatest returns a copy of the latest SML and Speedwire readings.
func (s *LiveStore) GetLatest() (SMLReading, SpeedwireReading) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latestSML, s.latestSpeedwire
}

// Subscribe returns a channel that is signaled on updates, and an unsubscribe function.
func (s *LiveStore) Subscribe() (chan struct{}, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan struct{}, 1)
	s.subscribers = append(s.subscribers, ch)

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, sub := range s.subscribers {
			if sub == ch {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsubscribe
}

func (s *LiveStore) notifySubscribers() {
	for _, ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
			// Buffer full, subscriber is already notified of a pending update
		}
	}
}

// PopHistory returns the accumulated SML and Speedwire history and clears the internal buffers.
// This is used by the database aggregator service every 5 minutes.
func (s *LiveStore) PopHistory() ([]SMLReading, []SpeedwireReading) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sml := s.smlHistory
	speedwire := s.speedwireHistory

	// Reset buffers with preallocated capacity for the next 5 minutes (approx 300 seconds)
	s.smlHistory = make([]SMLReading, 0, 300)
	s.speedwireHistory = make([]SpeedwireReading, 0, 300)

	return sml, speedwire
}
