package geometry

import (
	"fmt"
	"math"

	"github.com/cjeanneret/PanGo/internal/config"
)

// FOVCalculator computes field of view angles and required rotations
// based on lens and sensor configuration.
type FOVCalculator struct {
	cfg *config.Config
}

// NewFOVCalculator creates a new FOV calculator.
// Returns an error if sensor information is not available
// (required for calculations).
func NewFOVCalculator(cfg *config.Config) (*FOVCalculator, error) {
	if cfg.Sensor == nil {
		return nil, fmt.Errorf("sensor configuration is required for FOV calculations")
	}
	return &FOVCalculator{cfg: cfg}, nil
}

// HorizontalFOV calculates the horizontal field of view in degrees.
// Formula: FOV = 2 × arctan(sensor_width / (2 × focal_length))
func (f *FOVCalculator) HorizontalFOV() float64 {
	sensorWidth := f.cfg.Sensor.WidthMm
	focalLength := f.cfg.Lens.FocalLengthMm
	return 2.0 * math.Atan(sensorWidth/(2.0*focalLength)) * 180.0 / math.Pi
}

// VerticalFOV calculates the vertical field of view in degrees.
// Formula: FOV = 2 × arctan(sensor_height / (2 × focal_length))
func (f *FOVCalculator) VerticalFOV() float64 {
	sensorHeight := f.cfg.Sensor.HeightMm
	focalLength := f.cfg.Lens.FocalLengthMm
	return 2.0 * math.Atan(sensorHeight/(2.0*focalLength)) * 180.0 / math.Pi
}

// HorizontalRotationAngle calculates the horizontal rotation angle needed
// between two photos to achieve the desired overlap.
// If overlap = 30%, then each photo covers 70% new content.
// Angle = FOV_horizontal × (1 - overlap_ratio)
func (f *FOVCalculator) HorizontalRotationAngle() float64 {
	fov := f.HorizontalFOV()
	overlapRatio := f.cfg.OverlapRatio()
	return fov * (1.0 - overlapRatio)
}

// VerticalRotationAngle calculates the vertical rotation angle needed
// between two photos to achieve the desired overlap.
// If overlap = 30%, then each photo covers 70% new content.
// Angle = FOV_vertical × (1 - overlap_ratio)
func (f *FOVCalculator) VerticalRotationAngle() float64 {
	fov := f.VerticalFOV()
	overlapRatio := f.cfg.OverlapRatio()
	return fov * (1.0 - overlapRatio)
}
