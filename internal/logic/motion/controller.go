package motion

import "github.com/cjeanneret/PanGo/internal/hw/stepper"

// Controller orchestrates pan/tilt movements via two stepper motors.
// It's an intermediate layer between business logic (photo sequences,
// grids, scans, etc.) and low-level (GPIO).
type Controller struct {
	pan  *stepper.Stepper
	tilt *stepper.Stepper
}

func NewController(pan, tilt *stepper.Stepper) *Controller {
	return &Controller{
		pan:  pan,
		tilt: tilt,
	}
}

func (c *Controller) MovePan(steps int) error {
	return c.pan.MoveSteps(steps)
}

func (c *Controller) MoveTilt(steps int) error {
	return c.tilt.MoveSteps(steps)
}

// MovePanTilt performs a combined movement (sequential for now).
// Later, you can improve this method to synchronize the axes.
func (c *Controller) MovePanTilt(panSteps, tiltSteps int) error {
	if err := c.MovePan(panSteps); err != nil {
		return err
	}
	if err := c.MoveTilt(tiltSteps); err != nil {
		return err
	}
	return nil
}
