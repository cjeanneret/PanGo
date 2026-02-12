package capture

import (
	"context"
	"time"

	"github.com/cjeanneret/PanGo/internal/debug"
	"github.com/cjeanneret/PanGo/internal/hw/camera"
	"github.com/cjeanneret/PanGo/internal/logic/geometry"
	"github.com/cjeanneret/PanGo/internal/logic/motion"
)

// Sequence contains high-level logic for photo capture
// (grids, timelapse, panoramas, etc.).
type Sequence struct {
	motion *motion.Controller
	camera camera.Camera
}

func NewSequence(m *motion.Controller, c camera.Camera) *Sequence {
	return &Sequence{
		motion: m,
		camera: c,
	}
}

// GridShotParams defines the parameters for a grid traversal.
type GridShotParams struct {
	GridPlan *geometry.GridPlan // calculated grid plan

	Delay         time.Duration // delay between movements
	MoveSpeed     time.Duration // reserved for future improvements (ramping, etc.)
	ShotDelay     time.Duration // delay before shot (stabilization)
	PostShotDelay time.Duration // delay after shot before movement
}

// InitializePosition moves the head to the start position (far left, top).
func (s *Sequence) InitializePosition(plan *geometry.GridPlan) error {
	debug.Section("Initializing Position")
	debug.Live("Moving to start position (left, top)")

	// Go to start position from center (assuming we start from center)
	// First go left (negative pan)
	if plan.StartPanSteps != 0 {
		debug.Verbose("Moving pan: %d steps (to left)", plan.StartPanSteps)
		if err := s.motion.MovePan(plan.StartPanSteps); err != nil {
			return err
		}
	}

	// Then go up (positive tilt)
	if plan.StartTiltSteps != 0 {
		debug.Verbose("Moving tilt: %d steps (to top)", plan.StartTiltSteps)
		if err := s.motion.MoveTilt(plan.StartTiltSteps); err != nil {
			return err
		}
	}

	debug.Live("Initialization complete")
	return nil
}

// RunGridShot performs a grid traversal in columns (serpentine pattern):
// Column 0: top to bottom, then horizontal shift
// Column 1: bottom to top, then horizontal shift
// etc.
func (s *Sequence) RunGridShot(ctx context.Context, p GridShotParams) error {
	plan := p.GridPlan

	// Ensure motors are enabled before any movement
	_ = s.motion.EnableMotors()

	// Initialize: go to start position (left, top)
	if err := s.InitializePosition(plan); err != nil {
		return err
	}

	// Column traversal (serpentine)
	for col := 0; col < plan.PanColumns; col++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Determine vertical direction based on column (even = top->bottom, odd = bottom->top)
		goingDown := col%2 == 0
		direction := "up"
		if goingDown {
			direction = "down"
		}
		debug.Column(col+1, plan.PanColumns, direction)

		// Traverse column vertically
		for row := 0; row < plan.TiltRows; row++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// If not the first photo in the column, move vertically
			if row > 0 {
				// Vertical movement: always in the same direction based on column
				if goingDown {
					// Go down (negative tilt)
					debug.Move("tilt", plan.TiltStepSize, "down")
					if err := s.motion.MoveTilt(-plan.TiltStepSize); err != nil {
						return err
					}
				} else {
					// Go up (positive tilt)
					debug.Move("tilt", plan.TiltStepSize, "up")
					if err := s.motion.MoveTilt(plan.TiltStepSize); err != nil {
						return err
					}
				}
				time.Sleep(p.Delay)
			} else {
				debug.Verbose("  Row %d/%d: at start position", row+1, plan.TiltRows)
			}

			// Disable motors during capture (reduces vibration, no holding torque)
			_ = s.motion.DisableMotors()
			time.Sleep(p.ShotDelay)
			if err := s.camera.Shoot(); err != nil {
				_ = s.motion.EnableMotors()
				return err
			}
			debug.Shot(col+1, row+1)
			time.Sleep(p.PostShotDelay)
			// Re-enable motors for next movement
			_ = s.motion.EnableMotors()
		}

		// Horizontal shift to the right (except for the last column)
		if col < plan.PanColumns-1 {
			debug.Move("pan", plan.PanStepSize, "right")
			if err := s.motion.MovePan(plan.PanStepSize); err != nil {
				return err
			}
			time.Sleep(p.Delay)
		}
	}

	return nil
}
