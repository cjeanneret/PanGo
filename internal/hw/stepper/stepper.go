package stepper

import (
	"time"

	"github.com/cjeanneret/PanGo/internal/debug"
	"github.com/cjeanneret/PanGo/internal/hw/gpio"
)

// Config holds the hardware configuration for a stepper motor.
type Config struct {
	StepPin       int
	DirPin        int
	EnablePin     int           // A4988 ENABLE pin (BCM). 0 = not used. Active LOW (LOW=enabled).
	StepsPerRev   int
	Microstepping int
	StepDelay     time.Duration // delay per half-cycle of STEP pulse. Total step = 2*StepDelay.
}

// Stepper provides a simple API for moving a stepper motor.
// Acceleration, ramping, etc. can be added later.
type Stepper struct {
	gpio  gpio.Driver
	cfg   Config
	delay time.Duration // delay between STEP pulse half-cycles
}

// NewStepper creates a new stepper motor controller.
// cfg.StepDelay: if 0, defaults to 1ms. For A4988, use cfg.Defaults.MoveSpeedMs/2 per half-cycle.
func NewStepper(g gpio.Driver, cfg Config) *Stepper {
	_ = g.SetupPin(cfg.StepPin, gpio.Output)
	_ = g.SetupPin(cfg.DirPin, gpio.Output)

	delay := cfg.StepDelay
	if delay <= 0 {
		delay = 1 * time.Millisecond
	}

	s := &Stepper{
		gpio:  g,
		cfg:   cfg,
		delay: delay,
	}

	// A4988 ENABLE: active LOW. LOW = enabled, HIGH = disabled.
	if cfg.EnablePin > 0 {
		_ = g.SetupPin(cfg.EnablePin, gpio.Output)
		_ = g.WritePin(cfg.EnablePin, gpio.Low) // enable by default
	}

	return s
}

// MoveSteps moves the motor by a number of steps (positive or negative).
func (s *Stepper) MoveSteps(steps int) error {
	if steps == 0 {
		return nil
	}

	var dirLevel gpio.Level
	var direction string
	if steps > 0 {
		dirLevel = gpio.High
		direction = "forward"
	} else {
		dirLevel = gpio.Low
		direction = "backward"
		steps = -steps
	}

	debug.Printf("Stepper: moving %d steps (%s) on pin %d", steps, direction, s.cfg.StepPin)

	if err := s.gpio.WritePin(s.cfg.DirPin, dirLevel); err != nil {
		return err
	}

	for i := 0; i < steps; i++ {
		if err := s.stepPulse(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Stepper) stepPulse() error {
	if err := s.gpio.WritePin(s.cfg.StepPin, gpio.High); err != nil {
		return err
	}
	time.Sleep(s.delay)
	if err := s.gpio.WritePin(s.cfg.StepPin, gpio.Low); err != nil {
		return err
	}
	time.Sleep(s.delay)
	return nil
}

// Enable turns on the motor driver (A4988 ENABLE=LOW). Motors hold position.
func (s *Stepper) Enable() error {
	if s.cfg.EnablePin <= 0 {
		return nil
	}
	return s.gpio.WritePin(s.cfg.EnablePin, gpio.Low)
}

// Disable turns off the motor driver (A4988 ENABLE=HIGH). Motors freewheel, no holding torque.
// Use during photo capture to reduce vibration.
func (s *Stepper) Disable() error {
	if s.cfg.EnablePin <= 0 {
		return nil
	}
	return s.gpio.WritePin(s.cfg.EnablePin, gpio.High)
}
