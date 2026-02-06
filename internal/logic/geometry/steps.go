package geometry

import (
	"github.com/cjeanneret/PanGo/internal/config"
)

// StepsCalculator converts angles to motor step counts.
type StepsCalculator struct {
	panStepsPerDegree  float64
	tiltStepsPerDegree float64
}

// NewStepsCalculator creates a step calculator from configuration.
func NewStepsCalculator(cfg *config.Config) *StepsCalculator {
	// Calculate microsteps per degree for each axis
	panMicrostepsPerRev := float64(cfg.PanStepper.StepsPerRev * cfg.PanStepper.Microstepping)
	tiltMicrostepsPerRev := float64(cfg.TiltStepper.StepsPerRev * cfg.TiltStepper.Microstepping)

	panStepsPerDegree := panMicrostepsPerRev / 360.0
	tiltStepsPerDegree := tiltMicrostepsPerRev / 360.0

	return &StepsCalculator{
		panStepsPerDegree:  panStepsPerDegree,
		tiltStepsPerDegree: tiltStepsPerDegree,
	}
}

// PanStepsFromAngle converts a horizontal angle (in degrees) to motor steps.
func (s *StepsCalculator) PanStepsFromAngle(angleDegrees float64) int {
	return int(angleDegrees * s.panStepsPerDegree)
}

// TiltStepsFromAngle converts a vertical angle (in degrees) to motor steps.
func (s *StepsCalculator) TiltStepsFromAngle(angleDegrees float64) int {
	return int(angleDegrees * s.tiltStepsPerDegree)
}

// PanStepsForOverlap calculates the number of pan steps needed to achieve
// the configured overlap between two photos.
func (s *StepsCalculator) PanStepsForOverlap(fovCalc *FOVCalculator) int {
	angle := fovCalc.HorizontalRotationAngle()
	return s.PanStepsFromAngle(angle)
}

// TiltStepsForOverlap calculates the number of tilt steps needed to achieve
// the configured overlap between two photos.
func (s *StepsCalculator) TiltStepsForOverlap(fovCalc *FOVCalculator) int {
	angle := fovCalc.VerticalRotationAngle()
	return s.TiltStepsFromAngle(angle)
}
