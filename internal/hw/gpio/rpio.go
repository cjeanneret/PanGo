package gpio

import (
	"fmt"

	"github.com/cjeanneret/PanGo/internal/debug"
	"github.com/stianeikeland/go-rpio/v4"
)

// RPiDriver is the real implementation for Raspberry Pi using go-rpio.
type RPiDriver struct {
	pins map[int]rpio.Pin
}

// NewRPiRealDriver creates a real GPIO driver for Raspberry Pi.
// Requires running on a Raspberry Pi with access to /dev/gpiomem or as root.
func NewRPiRealDriver() (*RPiDriver, error) {
	debug.Info("Initializing real GPIO driver (go-rpio)")

	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("failed to open GPIO: %w (are you running on a Raspberry Pi?)", err)
	}

	debug.Verbose("GPIO memory mapped successfully")

	return &RPiDriver{
		pins: make(map[int]rpio.Pin),
	}, nil
}

func (r *RPiDriver) SetupPin(pin int, mode PinMode) error {
	debug.GPIO("SetupPin", pin, mode)

	p := rpio.Pin(pin)
	r.pins[pin] = p

	switch mode {
	case Input:
		p.Input()
	case Output:
		p.Output()
	default:
		return fmt.Errorf("unknown pin mode: %d", mode)
	}

	return nil
}

func (r *RPiDriver) WritePin(pin int, level Level) error {
	debug.GPIO("WritePin", pin, level)

	p, ok := r.pins[pin]
	if !ok {
		// Pin not setup yet, setup as output
		if err := r.SetupPin(pin, Output); err != nil {
			return err
		}
		p = r.pins[pin]
	}

	if level == High {
		p.High()
	} else {
		p.Low()
	}

	return nil
}

func (r *RPiDriver) ReadPin(pin int) (Level, error) {
	debug.GPIO("ReadPin", pin, nil)

	p, ok := r.pins[pin]
	if !ok {
		// Pin not setup yet, setup as input
		if err := r.SetupPin(pin, Input); err != nil {
			return Low, err
		}
		p = r.pins[pin]
	}

	state := p.Read()
	if state == rpio.High {
		return High, nil
	}
	return Low, nil
}

func (r *RPiDriver) Close() error {
	debug.Trace("GPIO Close (real driver)")

	// Reset all pins to input (safe state)
	for pin, p := range r.pins {
		debug.Verbose("Resetting pin %d to input", pin)
		p.Input()
	}

	return rpio.Close()
}
