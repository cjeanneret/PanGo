package geometry

import (
	"math"
	"testing"

	"github.com/cjeanneret/PanGo/internal/config"
)

func newGridConfig(focalMm, sensorW, sensorH, overlapPct, hAngle, vAngle float64) *config.Config {
	return &config.Config{
		Lens:   config.LensConfig{FocalLengthMm: focalMm},
		Sensor: &config.SensorConfig{WidthMm: sensorW, HeightMm: sensorH},
		PanStepper: config.StepperConfig{
			StepsPerRev: 200, Microstepping: 16,
		},
		TiltStepper: config.StepperConfig{
			StepsPerRev: 200, Microstepping: 16,
		},
		Defaults: config.DefaultsConfig{
			OverlapPercent:     overlapPct,
			HorizontalAngleDeg: hAngle,
			VerticalAngleDeg:   vAngle,
		},
	}
}

func TestCalculateGridPlan_StandardCase(t *testing.T) {
	cfg := newGridConfig(35, 23.6, 15.8, 30, 180, 30)
	fovCalc, _ := NewFOVCalculator(cfg)
	stepsCalc := NewStepsCalculator(cfg)

	plan := CalculateGridPlan(cfg, fovCalc, stepsCalc)

	// Verify basic properties
	if plan.PanColumns < 1 {
		t.Errorf("PanColumns = %d, must be >= 1", plan.PanColumns)
	}
	if plan.TiltRows < 1 {
		t.Errorf("TiltRows = %d, must be >= 1", plan.TiltRows)
	}

	// Verify calculated values match expected formulas
	panRotation := fovCalc.HorizontalRotationAngle()
	tiltRotation := fovCalc.VerticalRotationAngle()
	expectedCols := int(math.Ceil(180.0 / panRotation))
	expectedRows := int(math.Ceil(30.0 / tiltRotation))

	if plan.PanColumns != expectedCols {
		t.Errorf("PanColumns = %d, want %d", plan.PanColumns, expectedCols)
	}
	if plan.TiltRows != expectedRows {
		t.Errorf("TiltRows = %d, want %d", plan.TiltRows, expectedRows)
	}

	// Verify start angles
	if math.Abs(plan.StartPanAngle-(-90.0)) > epsilon {
		t.Errorf("StartPanAngle = %v, want -90.0", plan.StartPanAngle)
	}
	if math.Abs(plan.StartTiltAngle-15.0) > epsilon {
		t.Errorf("StartTiltAngle = %v, want 15.0", plan.StartTiltAngle)
	}

	// Step sizes must be positive
	if plan.PanStepSize <= 0 {
		t.Errorf("PanStepSize = %d, must be > 0", plan.PanStepSize)
	}
	if plan.TiltStepSize <= 0 {
		t.Errorf("TiltStepSize = %d, must be > 0", plan.TiltStepSize)
	}
}

func TestCalculateGridPlan_MinimumGrid(t *testing.T) {
	// Angle smaller than FOV -> 1 column, 1 row
	cfg := newGridConfig(35, 23.6, 15.8, 30, 5, 5)
	fovCalc, _ := NewFOVCalculator(cfg)
	stepsCalc := NewStepsCalculator(cfg)

	plan := CalculateGridPlan(cfg, fovCalc, stepsCalc)

	if plan.PanColumns < 1 {
		t.Errorf("PanColumns = %d, must be >= 1", plan.PanColumns)
	}
	if plan.TiltRows < 1 {
		t.Errorf("TiltRows = %d, must be >= 1", plan.TiltRows)
	}
}

func TestCalculateGridPlan_FullPanorama360(t *testing.T) {
	cfg := newGridConfig(35, 23.6, 15.8, 30, 360, 30)
	fovCalc, _ := NewFOVCalculator(cfg)
	stepsCalc := NewStepsCalculator(cfg)

	plan := CalculateGridPlan(cfg, fovCalc, stepsCalc)

	// 360 deg / rotation_angle should give many columns
	if plan.PanColumns < 10 {
		t.Errorf("360 panorama should have many columns, got %d", plan.PanColumns)
	}
}

func TestCalculateGridPlan_LargeOverlap(t *testing.T) {
	cfg90 := newGridConfig(35, 23.6, 15.8, 90, 180, 30)
	cfg5 := newGridConfig(35, 23.6, 15.8, 5, 180, 30)

	fov90, _ := NewFOVCalculator(cfg90)
	steps90 := NewStepsCalculator(cfg90)
	plan90 := CalculateGridPlan(cfg90, fov90, steps90)

	fov5, _ := NewFOVCalculator(cfg5)
	steps5 := NewStepsCalculator(cfg5)
	plan5 := CalculateGridPlan(cfg5, fov5, steps5)

	if plan90.PanColumns <= plan5.PanColumns {
		t.Errorf("90%% overlap columns (%d) should be more than 5%% overlap columns (%d)",
			plan90.PanColumns, plan5.PanColumns)
	}
	if plan90.TiltRows <= plan5.TiltRows {
		t.Errorf("90%% overlap rows (%d) should be more than 5%% overlap rows (%d)",
			plan90.TiltRows, plan5.TiltRows)
	}
}

func TestCalculateGridPlan_StartPositions(t *testing.T) {
	cases := []struct {
		name      string
		hAngle    float64
		vAngle    float64
		wantPanA  float64
		wantTiltA float64
	}{
		{"180x30", 180, 30, -90, 15},
		{"360x60", 360, 60, -180, 30},
		{"90x10", 90, 10, -45, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newGridConfig(35, 23.6, 15.8, 30, tc.hAngle, tc.vAngle)
			fovCalc, _ := NewFOVCalculator(cfg)
			stepsCalc := NewStepsCalculator(cfg)
			plan := CalculateGridPlan(cfg, fovCalc, stepsCalc)

			if math.Abs(plan.StartPanAngle-tc.wantPanA) > epsilon {
				t.Errorf("StartPanAngle = %v, want %v", plan.StartPanAngle, tc.wantPanA)
			}
			if math.Abs(plan.StartTiltAngle-tc.wantTiltA) > epsilon {
				t.Errorf("StartTiltAngle = %v, want %v", plan.StartTiltAngle, tc.wantTiltA)
			}
		})
	}
}

func TestCalculateGridPlan_StepSizesPositive(t *testing.T) {
	configs := []struct {
		name   string
		focal  float64
		hAngle float64
		vAngle float64
	}{
		{"wide", 18, 180, 60},
		{"normal", 50, 180, 30},
		{"tele", 200, 90, 15},
	}
	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newGridConfig(tc.focal, 23.6, 15.8, 30, tc.hAngle, tc.vAngle)
			fovCalc, _ := NewFOVCalculator(cfg)
			stepsCalc := NewStepsCalculator(cfg)
			plan := CalculateGridPlan(cfg, fovCalc, stepsCalc)

			if plan.PanStepSize <= 0 {
				t.Errorf("PanStepSize = %d, must be > 0", plan.PanStepSize)
			}
			if plan.TiltStepSize <= 0 {
				t.Errorf("TiltStepSize = %d, must be > 0", plan.TiltStepSize)
			}
		})
	}
}

func TestCalculateGridPlan_AlwaysAtLeastOnePhoto(t *testing.T) {
	// Even with very small angles, should have at least 1x1
	cfg := newGridConfig(35, 23.6, 15.8, 30, 0.1, 0.1)
	fovCalc, _ := NewFOVCalculator(cfg)
	stepsCalc := NewStepsCalculator(cfg)
	plan := CalculateGridPlan(cfg, fovCalc, stepsCalc)

	if plan.PanColumns < 1 {
		t.Errorf("PanColumns = %d, must be >= 1", plan.PanColumns)
	}
	if plan.TiltRows < 1 {
		t.Errorf("TiltRows = %d, must be >= 1", plan.TiltRows)
	}
}
