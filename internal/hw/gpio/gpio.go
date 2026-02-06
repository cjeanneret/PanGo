package gpio

import (
	"github.com/cjeanneret/PanGo/internal/debug"
)

// Level represents the logical state of a GPIO pin.
type Level bool

const (
	Low  Level = false
	High Level = true
)

// PinMode indicates whether a GPIO is input or output.
type PinMode int

const (
	Input PinMode = iota
	Output
)

// Driver defines the abstract interface for controlling GPIOs.
// This allows plugging in a real Raspberry Pi implementation
// or a mock for development on PC.
type Driver interface {
	SetupPin(pin int, mode PinMode) error
	WritePin(pin int, level Level) error
	ReadPin(pin int) (Level, error)
	Close() error
}

// MockDriver is a test implementation that simply logs actions.
// Used for development on PC or testing.
type MockDriver struct{}

// NewDriver creates a GPIO driver based on the chosen mode.
// If mock is true, returns a MockDriver (for dev/test).
// If mock is false, returns a real RPiDriver (for Raspberry Pi).
func NewDriver(mock bool) (Driver, error) {
	if mock {
		debug.Info("Using MOCK GPIO driver (development mode)")
		return &MockDriver{}, nil
	}
	return NewRPiRealDriver()
}

func (m *MockDriver) SetupPin(pin int, mode PinMode) error {
	debug.GPIO("SetupPin", pin, mode)
	return nil
}

func (m *MockDriver) WritePin(pin int, level Level) error {
	debug.GPIO("WritePin", pin, level)
	return nil
}

func (m *MockDriver) ReadPin(pin int) (Level, error) {
	debug.GPIO("ReadPin", pin, nil)
	return Low, nil
}

func (m *MockDriver) Close() error {
	debug.Trace("GPIO Close (mock)")
	return nil
}
