package motion

import (
	"testing"
	"time"

	"github.com/cjeanneret/PanGo/internal/hw/gpio"
	"github.com/cjeanneret/PanGo/internal/hw/stepper"
)

func newMockStepper() (*stepper.Stepper, *gpio.MockDriver) {
	drv := &gpio.MockDriver{}
	s := stepper.NewStepper(drv, stepper.Config{
		StepPin:       1,
		DirPin:        2,
		EnablePin:     3,
		StepsPerRev:   200,
		Microstepping: 16,
		StepDelay:     1 * time.Microsecond,
	})
	return s, drv
}

func TestController_MovePan(t *testing.T) {
	pan, _ := newMockStepper()
	tilt, _ := newMockStepper()
	ctrl := NewController(pan, tilt)

	if err := ctrl.MovePan(100); err != nil {
		t.Errorf("MovePan: %v", err)
	}
}

func TestController_MoveTilt(t *testing.T) {
	pan, _ := newMockStepper()
	tilt, _ := newMockStepper()
	ctrl := NewController(pan, tilt)

	if err := ctrl.MoveTilt(50); err != nil {
		t.Errorf("MoveTilt: %v", err)
	}
}

func TestController_MovePanTilt(t *testing.T) {
	pan, _ := newMockStepper()
	tilt, _ := newMockStepper()
	ctrl := NewController(pan, tilt)

	if err := ctrl.MovePanTilt(100, 50); err != nil {
		t.Errorf("MovePanTilt: %v", err)
	}
}

func TestController_EnableMotors(t *testing.T) {
	pan, _ := newMockStepper()
	tilt, _ := newMockStepper()
	ctrl := NewController(pan, tilt)

	if err := ctrl.EnableMotors(); err != nil {
		t.Errorf("EnableMotors: %v", err)
	}
}

func TestController_DisableMotors(t *testing.T) {
	pan, _ := newMockStepper()
	tilt, _ := newMockStepper()
	ctrl := NewController(pan, tilt)

	if err := ctrl.DisableMotors(); err != nil {
		t.Errorf("DisableMotors: %v", err)
	}
}

func TestController_MovePanZero(t *testing.T) {
	pan, _ := newMockStepper()
	tilt, _ := newMockStepper()
	ctrl := NewController(pan, tilt)

	if err := ctrl.MovePan(0); err != nil {
		t.Errorf("MovePan(0): %v", err)
	}
}
