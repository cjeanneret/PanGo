package stepper

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
	op    string // "setup", "write"
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

func (d *recordingDriver) Close() error {
	return nil
}

func (d *recordingDriver) writeCalls() []gpioCall {
	var result []gpioCall
	for _, c := range d.calls {
		if c.op == "write" {
			result = append(result, c)
		}
	}
	return result
}

func (d *recordingDriver) writeCallsForPin(pin int) []gpioCall {
	var result []gpioCall
	for _, c := range d.calls {
		if c.op == "write" && c.pin == pin {
			result = append(result, c)
		}
	}
	return result
}

func TestStepper_MoveStepsForward(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		EnablePin:     5,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	}
	s := NewStepper(drv, cfg)
	drv.calls = nil // reset after init

	if err := s.MoveSteps(10); err != nil {
		t.Fatalf("MoveSteps: %v", err)
	}

	// First call should set direction HIGH (forward)
	writes := drv.writeCalls()
	if len(writes) == 0 {
		t.Fatal("expected GPIO write calls")
	}
	if writes[0].pin != 27 || writes[0].level != gpio.High {
		t.Errorf("first write should set dir pin HIGH, got pin=%d level=%v", writes[0].pin, writes[0].level)
	}

	// Count step pulses (HIGH+LOW pairs on step pin)
	stepPulses := 0
	for _, c := range writes {
		if c.pin == cfg.StepPin && c.level == gpio.High {
			stepPulses++
		}
	}
	if stepPulses != 10 {
		t.Errorf("expected 10 step pulses, got %d", stepPulses)
	}
}

func TestStepper_MoveStepsBackward(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		EnablePin:     5,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	}
	s := NewStepper(drv, cfg)
	drv.calls = nil

	if err := s.MoveSteps(-5); err != nil {
		t.Fatalf("MoveSteps: %v", err)
	}

	writes := drv.writeCalls()
	if len(writes) == 0 {
		t.Fatal("expected GPIO write calls")
	}
	// Direction should be LOW (backward)
	if writes[0].pin != 27 || writes[0].level != gpio.Low {
		t.Errorf("first write should set dir pin LOW, got pin=%d level=%v", writes[0].pin, writes[0].level)
	}

	stepPulses := 0
	for _, c := range writes {
		if c.pin == cfg.StepPin && c.level == gpio.High {
			stepPulses++
		}
	}
	if stepPulses != 5 {
		t.Errorf("expected 5 step pulses, got %d", stepPulses)
	}
}

func TestStepper_MoveStepsZero(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		EnablePin:     5,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	}
	s := NewStepper(drv, cfg)
	drv.calls = nil

	if err := s.MoveSteps(0); err != nil {
		t.Fatalf("MoveSteps: %v", err)
	}

	if len(drv.calls) != 0 {
		t.Errorf("zero steps should produce no GPIO calls, got %d", len(drv.calls))
	}
}

func TestStepper_EnableDisable(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		EnablePin:     5,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	}
	s := NewStepper(drv, cfg)
	drv.calls = nil

	if err := s.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	enableCalls := drv.writeCallsForPin(5)
	if len(enableCalls) != 1 || enableCalls[0].level != gpio.Low {
		t.Errorf("Enable should write LOW to enable pin, got %v", enableCalls)
	}

	drv.calls = nil
	if err := s.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	disableCalls := drv.writeCallsForPin(5)
	if len(disableCalls) != 1 || disableCalls[0].level != gpio.High {
		t.Errorf("Disable should write HIGH to enable pin, got %v", disableCalls)
	}
}

func TestStepper_EnableDisable_NoEnablePin(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		EnablePin:     0, // no enable pin
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	}
	s := NewStepper(drv, cfg)
	drv.calls = nil

	if err := s.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if err := s.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	if len(drv.calls) != 0 {
		t.Errorf("with EnablePin=0, Enable/Disable should produce no GPIO calls, got %d", len(drv.calls))
	}
}

func TestStepper_DefaultStepDelay(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     0, // should default to 1ms
	}
	s := NewStepper(drv, cfg)
	if s.delay != 1*time.Millisecond {
		t.Errorf("default delay = %v, want 1ms", s.delay)
	}
}

func TestStepper_StepPulsePattern(t *testing.T) {
	drv := &recordingDriver{}
	cfg := Config{
		StepPin:       17,
		DirPin:        27,
		EnablePin:     5,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	}
	s := NewStepper(drv, cfg)
	drv.calls = nil

	s.MoveSteps(1) // single step

	stepCalls := drv.writeCallsForPin(17)
	// Should be HIGH then LOW
	if len(stepCalls) != 2 {
		t.Fatalf("single step should produce 2 writes on step pin, got %d", len(stepCalls))
	}
	if stepCalls[0].level != gpio.High {
		t.Error("first pulse should be HIGH")
	}
	if stepCalls[1].level != gpio.Low {
		t.Error("second pulse should be LOW")
	}
}
