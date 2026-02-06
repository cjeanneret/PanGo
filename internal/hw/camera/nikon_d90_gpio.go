package camera

import (
	"time"

	"github.com/cjeanneret/PanGo/internal/debug"
	"github.com/cjeanneret/PanGo/internal/hw/gpio"
)

// NikonD90GPIO is a Camera implementation for a Nikon D90
// controlled via the 3-pin remote connector:
// - GND: connected to Raspberry Pi ground
// - FOCUS: autofocus (activate by setting to LOW)
// - SHUTTER: trigger (activate by setting to LOW)
//
// Trigger sequence:
// 1. FOCUS to LOW (activates autofocus)
// 2. Wait for autofocus to complete
// 3. SHUTTER to LOW (triggers the shot)
// 4. Hold for a moment
// 5. Set SHUTTER and FOCUS back to HIGH
type NikonD90GPIO struct {
	gpio         gpio.Driver
	focusPin     int
	shutterPin   int
	focusDelay   time.Duration // time for autofocus
	shutterDelay time.Duration // shutter hold time
}

// NewNikonD90GPIO creates a GPIO-controlled Nikon D90 trigger.
// focusPin and shutterPin are the GPIO pin numbers for FOCUS and SHUTTER lines.
// focusDelay is the wait time for autofocus.
// shutterDelay is the shutter hold time.
func NewNikonD90GPIO(g gpio.Driver, focusPin, shutterPin int, focusDelay, shutterDelay time.Duration) *NikonD90GPIO {
	// Configure pins as outputs
	_ = g.SetupPin(focusPin, gpio.Output)
	_ = g.SetupPin(shutterPin, gpio.Output)

	// By default, lines are HIGH (inactive)
	_ = g.WritePin(focusPin, gpio.High)
	_ = g.WritePin(shutterPin, gpio.High)

	return &NikonD90GPIO{
		gpio:         g,
		focusPin:     focusPin,
		shutterPin:   shutterPin,
		focusDelay:   focusDelay,
		shutterDelay: shutterDelay,
	}
}

// Shoot triggers a photo on the D90.
// Sequence: FOCUS -> wait for AF -> SHUTTER -> hold -> release
func (n *NikonD90GPIO) Shoot() error {
	debug.Printf("Camera: triggering shot (focus=%d, shutter=%d)", n.focusPin, n.shutterPin)

	// 1. Activate FOCUS (autofocus)
	debug.Verbose("Camera: activating FOCUS (pin %d -> LOW)", n.focusPin)
	if err := n.gpio.WritePin(n.focusPin, gpio.Low); err != nil {
		return err
	}

	// 2. Wait for autofocus to complete
	debug.Verbose("Camera: waiting for autofocus (%v)", n.focusDelay)
	time.Sleep(n.focusDelay)

	// 3. Activate SHUTTER (trigger)
	debug.Verbose("Camera: activating SHUTTER (pin %d -> LOW)", n.shutterPin)
	if err := n.gpio.WritePin(n.shutterPin, gpio.Low); err != nil {
		// Release FOCUS on error
		_ = n.gpio.WritePin(n.focusPin, gpio.High)
		return err
	}

	// 4. Hold shutter
	debug.Verbose("Camera: holding shutter (%v)", n.shutterDelay)
	time.Sleep(n.shutterDelay)

	// 5. Release SHUTTER then FOCUS
	debug.Verbose("Camera: releasing SHUTTER (pin %d -> HIGH)", n.shutterPin)
	if err := n.gpio.WritePin(n.shutterPin, gpio.High); err != nil {
		return err
	}

	debug.Verbose("Camera: releasing FOCUS (pin %d -> HIGH)", n.focusPin)
	if err := n.gpio.WritePin(n.focusPin, gpio.High); err != nil {
		return err
	}

	debug.Print("Camera: shot triggered successfully")
	return nil
}
