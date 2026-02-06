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
	StepsPerRev   int
	Microstepping int
}

// Stepper provides a simple API for moving a stepper motor.
// Acceleration, ramping, etc. can be added later.
type Stepper struct {
	gpio  gpio.Driver
	cfg   Config
	delay time.Duration // delay between STEP pulses
}

// NewStepper creates a new stepper motor controller.
func NewStepper(g gpio.Driver, cfg Config) *Stepper {
	// TODO: handle delay configuration via config (speed, acceleration, etc.)
	_ = g.SetupPin(cfg.StepPin, gpio.Output)
	_ = g.SetupPin(cfg.DirPin, gpio.Output)

	return &Stepper{
		gpio:  g,
		cfg:   cfg,
		delay: 1 * time.Millisecond,
	}
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
