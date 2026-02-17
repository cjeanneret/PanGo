package geometry

import (
	"testing"

	"github.com/cjeanneret/PanGo/internal/config"
)

func newStepsConfig(stepsPerRev, microstepping int) *config.Config {
	return &config.Config{
		PanStepper: config.StepperConfig{
			StepsPerRev:   stepsPerRev,
			Microstepping: microstepping,
		},
		TiltStepper: config.StepperConfig{
			StepsPerRev:   stepsPerRev,
			Microstepping: microstepping,
		},
	}
}

func TestStepsCalculator_KnownConfig(t *testing.T) {
	// 200 steps/rev * 16 microstepping = 3200 microsteps/rev
	// stepsPerDegree = 3200 / 360 ≈ 8.888...
	cfg := newStepsConfig(200, 16)
	sc := NewStepsCalculator(cfg)

	spd := 3200.0 / 360.0 // steps per degree
	cases := []struct {
		name  string
		angle float64
		want  int
	}{
		{"90_degrees", 90, int(90 * spd)},
		{"negative_90", -90, int(-90 * spd)},
		{"zero", 0, 0},
		{"full_360", 360, int(360 * spd)},
		{"small_1_degree", 1, int(1 * spd)},
	}
	for _, tc := range cases {
		t.Run("Pan_"+tc.name, func(t *testing.T) {
			got := sc.PanStepsFromAngle(tc.angle)
			if got != tc.want {
				t.Errorf("PanStepsFromAngle(%v) = %d, want %d", tc.angle, got, tc.want)
			}
		})
		t.Run("Tilt_"+tc.name, func(t *testing.T) {
			got := sc.TiltStepsFromAngle(tc.angle)
			if got != tc.want {
				t.Errorf("TiltStepsFromAngle(%v) = %d, want %d", tc.angle, got, tc.want)
			}
		})
	}
}

func TestStepsCalculator_DifferentMicrostepping(t *testing.T) {
	microsteps := []int{1, 2, 4, 8, 16, 32}
	for _, ms := range microsteps {
		cfg := newStepsConfig(200, ms)
		sc := NewStepsCalculator(cfg)
		microstepsPerRev := float64(200 * ms)
		want := int(90.0 * microstepsPerRev / 360.0)
		got := sc.PanStepsFromAngle(90)
		if got != want {
			t.Errorf("microstepping=%d: PanStepsFromAngle(90) = %d, want %d", ms, got, want)
		}
	}
}

func TestStepsCalculator_AsymmetricAxes(t *testing.T) {
	cfg := &config.Config{
		PanStepper: config.StepperConfig{
			StepsPerRev:   200,
			Microstepping: 16,
		},
		TiltStepper: config.StepperConfig{
			StepsPerRev:   400,
			Microstepping: 8,
		},
	}
	sc := NewStepsCalculator(cfg)

	panSteps := sc.PanStepsFromAngle(90)
	tiltSteps := sc.TiltStepsFromAngle(90)

	panExpected := int(90.0 * float64(200*16) / 360.0)
	tiltExpected := int(90.0 * float64(400*8) / 360.0)

	if panSteps != panExpected {
		t.Errorf("pan steps = %d, want %d", panSteps, panExpected)
	}
	if tiltSteps != tiltExpected {
		t.Errorf("tilt steps = %d, want %d", tiltSteps, tiltExpected)
	}
}

func TestStepsCalculator_ForOverlap(t *testing.T) {
	cfg := &config.Config{
		Lens:   config.LensConfig{FocalLengthMm: 35},
		Sensor: &config.SensorConfig{WidthMm: 23.6, HeightMm: 15.8},
		PanStepper: config.StepperConfig{
			StepsPerRev: 200, Microstepping: 16,
		},
		TiltStepper: config.StepperConfig{
			StepsPerRev: 200, Microstepping: 16,
		},
		Defaults: config.DefaultsConfig{OverlapPercent: 30},
	}

	fovCalc, err := NewFOVCalculator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sc := NewStepsCalculator(cfg)

	panSteps := sc.PanStepsForOverlap(fovCalc)
	tiltSteps := sc.TiltStepsForOverlap(fovCalc)

	expectedPan := sc.PanStepsFromAngle(fovCalc.HorizontalRotationAngle())
	expectedTilt := sc.TiltStepsFromAngle(fovCalc.VerticalRotationAngle())

	if panSteps != expectedPan {
		t.Errorf("PanStepsForOverlap = %d, want %d", panSteps, expectedPan)
	}
	if tiltSteps != expectedTilt {
		t.Errorf("TiltStepsForOverlap = %d, want %d", tiltSteps, expectedTilt)
	}
}
