package geometry

import (
	"math"

	"github.com/cjeanneret/PanGo/internal/config"
)

// GridPlan calculates the photo grid plan needed
// to cover the total angle with the desired overlap.
type GridPlan struct {
	PanColumns   int // number of columns (horizontal photos)
	TiltRows     int // number of rows (vertical photos)
	PanStepSize  int // motor steps between each photo horizontally
	TiltStepSize int // motor steps between each photo vertically

	// Start positions (from center)
	StartPanAngle  float64 // starting pan angle (left)
	StartTiltAngle float64 // starting tilt angle (top)

	// Motor steps to reach start position
	StartPanSteps  int // motor steps to go left from center
	StartTiltSteps int // motor steps to go up from center
}

// CalculateGridPlan calculates the complete grid plan from config
// and FOV/steps calculators.
func CalculateGridPlan(cfg *config.Config, fovCalc *FOVCalculator, stepsCalc *StepsCalculator) *GridPlan {
	// Rotation angles between each photo
	panRotationAngle := fovCalc.HorizontalRotationAngle()
	tiltRotationAngle := fovCalc.VerticalRotationAngle()

	// Total angles to cover
	totalPanAngle := cfg.HorizontalAngleDeg()
	totalTiltAngle := cfg.VerticalAngleDeg()

	// Calculate number of photos needed
	// Round up to ensure we cover the entire angle
	panColumns := int(math.Ceil(totalPanAngle / panRotationAngle))
	tiltRows := int(math.Ceil(totalTiltAngle / tiltRotationAngle))

	// Ensure at least 1 photo if needed
	if panColumns < 1 {
		panColumns = 1
	}
	if tiltRows < 1 {
		tiltRows = 1
	}

	// Convert to motor steps
	panStepSize := stepsCalc.PanStepsFromAngle(panRotationAngle)
	tiltStepSize := stepsCalc.TiltStepsFromAngle(tiltRotationAngle)

	// Start position: far left (negative) and top (positive)
	// Note: we assume "up" = positive angle for tilt
	startPanAngle := -cfg.HorizontalHalfAngleDeg()  // left
	startTiltAngle := cfg.VerticalHalfAngleDeg()    // top

	startPanSteps := stepsCalc.PanStepsFromAngle(startPanAngle)
	startTiltSteps := stepsCalc.TiltStepsFromAngle(startTiltAngle)

	return &GridPlan{
		PanColumns:     panColumns,
		TiltRows:       tiltRows,
		PanStepSize:    panStepSize,
		TiltStepSize:   tiltStepSize,
		StartPanAngle:  startPanAngle,
		StartTiltAngle: startTiltAngle,
		StartPanSteps:  startPanSteps,
		StartTiltSteps: startTiltSteps,
	}
}
