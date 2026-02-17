package camera

import (
	"testing"
	"time"

	"github.com/cjeanneret/PanGo/internal/hw/gpio"
)

// recordingDriver records GPIO calls for verification.
type recordingDriver struct {
	calls []gpioCall
}

type gpioCall struct {
	op    string
	pin   int
	level gpio.Level
}

func (d *recordingDriver) SetupPin(pin int, mode gpio.PinMode) error {
	d.calls = append(d.calls, gpioCall{op: "setup", pin: pin})
	return nil
}

func (d *recordingDriver) WritePin(pin int, level gpio.Level) error {
	d.calls = append(d.calls, gpioCall{op: "write", pin: pin, level: level})
	return nil
}

func (d *recordingDriver) ReadPin(pin int) (gpio.Level, error) {
	return gpio.Low, nil
}

func (d *recordingDriver) Close() error { return nil }

func (d *recordingDriver) writeCalls() []gpioCall {
	var result []gpioCall
	for _, c := range d.calls {
		if c.op == "write" {
			result = append(result, c)
		}
	}
	return result
}

func TestNikonD90GPIO_PinsInitializedHigh(t *testing.T) {
	drv := &recordingDriver{}
	NewNikonD90GPIO(drv, 24, 25, 500*time.Millisecond, 200*time.Millisecond)

	// After construction, both pins should have been set to HIGH (inactive)
	writes := drv.writeCalls()
	focusHigh := false
	shutterHigh := false
	for _, c := range writes {
		if c.pin == 24 && c.level == gpio.High {
			focusHigh = true
		}
		if c.pin == 25 && c.level == gpio.High {
			shutterHigh = true
		}
	}
	if !focusHigh {
		t.Error("focus pin should be initialized to HIGH")
	}
	if !shutterHigh {
		t.Error("shutter pin should be initialized to HIGH")
	}
}

func TestNikonD90GPIO_ShootSequence(t *testing.T) {
	drv := &recordingDriver{}
	cam := NewNikonD90GPIO(drv, 24, 25, 1*time.Microsecond, 1*time.Microsecond)
	drv.calls = nil // reset after init

	if err := cam.Shoot(); err != nil {
		t.Fatalf("Shoot: %v", err)
	}

	writes := drv.writeCalls()
	// Expected sequence:
	// 1. Focus LOW (activate autofocus)
	// 2. Shutter LOW (trigger)
	// 3. Shutter HIGH (release)
	// 4. Focus HIGH (release)

	expected := []struct {
		pin   int
		level gpio.Level
		desc  string
	}{
		{24, gpio.Low, "focus LOW (activate AF)"},
		{25, gpio.Low, "shutter LOW (trigger)"},
		{25, gpio.High, "shutter HIGH (release)"},
		{24, gpio.High, "focus HIGH (release)"},
	}

	if len(writes) != len(expected) {
		t.Fatalf("expected %d writes, got %d: %v", len(expected), len(writes), writes)
	}

	for i, exp := range expected {
		if writes[i].pin != exp.pin || writes[i].level != exp.level {
			t.Errorf("step %d (%s): pin=%d level=%v, want pin=%d level=%v",
				i, exp.desc, writes[i].pin, writes[i].level, exp.pin, exp.level)
		}
	}
}

func TestNikonD90GPIO_ShootReturnsNoError(t *testing.T) {
	drv := &recordingDriver{}
	cam := NewNikonD90GPIO(drv, 24, 25, 1*time.Microsecond, 1*time.Microsecond)
	if err := cam.Shoot(); err != nil {
		t.Errorf("Shoot should not error with mock driver, got: %v", err)
	}
}

func TestNikonD90GPIO_ImplementsCamera(t *testing.T) {
	drv := &recordingDriver{}
	cam := NewNikonD90GPIO(drv, 24, 25, time.Millisecond, time.Millisecond)
	var _ Camera = cam // compile-time check
}
