package geometry

import (
	"math"
	"testing"

	"github.com/cjeanneret/PanGo/internal/config"
)

const epsilon = 0.01 // tolerance for float comparisons (degrees)

func newFOVConfig(focalMm, sensorW, sensorH, overlapPct float64) *config.Config {
	return &config.Config{
		Lens:   config.LensConfig{FocalLengthMm: focalMm},
		Sensor: &config.SensorConfig{WidthMm: sensorW, HeightMm: sensorH},
		Defaults: config.DefaultsConfig{
			OverlapPercent: overlapPct,
		},
	}
}

func TestNewFOVCalculator_NilSensor(t *testing.T) {
	cfg := &config.Config{Lens: config.LensConfig{FocalLengthMm: 35}}
	_, err := NewFOVCalculator(cfg)
	if err == nil {
		t.Error("expected error for nil sensor, got nil")
	}
}

func TestNewFOVCalculator_ValidSensor(t *testing.T) {
	cfg := newFOVConfig(35, 23.6, 15.8, 30)
	fov, err := NewFOVCalculator(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fov == nil {
		t.Fatal("expected non-nil calculator")
	}
}

// Reference: Nikon APS-C (23.6 x 15.8 mm) with 35mm lens
// HorizontalFOV = 2 * atan(23.6 / (2*35)) * 180/pi ~ 37.22 deg
// VerticalFOV   = 2 * atan(15.8 / (2*35)) * 180/pi ~ 25.43 deg
func TestFOVCalculator_HorizontalFOV_NikonAPSC_35mm(t *testing.T) {
	cfg := newFOVConfig(35, 23.6, 15.8, 30)
	fov, _ := NewFOVCalculator(cfg)

	got := fov.HorizontalFOV()
	want := 2.0 * math.Atan(23.6/(2.0*35.0)) * 180.0 / math.Pi

	if math.Abs(got-want) > epsilon {
		t.Errorf("HorizontalFOV() = %v, want ~%v", got, want)
	}
}

func TestFOVCalculator_VerticalFOV_NikonAPSC_35mm(t *testing.T) {
	cfg := newFOVConfig(35, 23.6, 15.8, 30)
	fov, _ := NewFOVCalculator(cfg)

	got := fov.VerticalFOV()
	want := 2.0 * math.Atan(15.8/(2.0*35.0)) * 180.0 / math.Pi

	if math.Abs(got-want) > epsilon {
		t.Errorf("VerticalFOV() = %v, want ~%v", got, want)
	}
}

func TestFOVCalculator_FOV_DecreasesWithFocalLength(t *testing.T) {
	cfg18 := newFOVConfig(18, 23.6, 15.8, 30)
	cfg200 := newFOVConfig(200, 23.6, 15.8, 30)
	fov18, _ := NewFOVCalculator(cfg18)
	fov200, _ := NewFOVCalculator(cfg200)

	if fov18.HorizontalFOV() <= fov200.HorizontalFOV() {
		t.Errorf("18mm FOV (%v) should be larger than 200mm FOV (%v)",
			fov18.HorizontalFOV(), fov200.HorizontalFOV())
	}
	if fov18.VerticalFOV() <= fov200.VerticalFOV() {
		t.Errorf("18mm vertical FOV (%v) should be larger than 200mm vertical FOV (%v)",
			fov18.VerticalFOV(), fov200.VerticalFOV())
	}
}

func TestFOVCalculator_RotationAngle_Overlap30(t *testing.T) {
	cfg := newFOVConfig(35, 23.6, 15.8, 30)
	fov, _ := NewFOVCalculator(cfg)

	hFOV := fov.HorizontalFOV()
	wantH := hFOV * 0.7
	gotH := fov.HorizontalRotationAngle()
	if math.Abs(gotH-wantH) > epsilon {
		t.Errorf("HorizontalRotationAngle() = %v, want %v (FOV*0.7)", gotH, wantH)
	}

	vFOV := fov.VerticalFOV()
	wantV := vFOV * 0.7
	gotV := fov.VerticalRotationAngle()
	if math.Abs(gotV-wantV) > epsilon {
		t.Errorf("VerticalRotationAngle() = %v, want %v (FOV*0.7)", gotV, wantV)
	}
}

func TestFOVCalculator_RotationAngle_ZeroOverlap(t *testing.T) {
	cfg := newFOVConfig(35, 23.6, 15.8, 0)
	// overlap 0 defaults to 30 in Load, but we set it directly here
	// OverlapRatio() = 0/100 = 0, so rotation = FOV * 1.0
	fov, _ := NewFOVCalculator(cfg)

	hFOV := fov.HorizontalFOV()
	gotH := fov.HorizontalRotationAngle()
	if math.Abs(gotH-hFOV) > epsilon {
		t.Errorf("with 0%% overlap, rotation (%v) should equal FOV (%v)", gotH, hFOV)
	}
}

func TestFOVCalculator_RotationAngle_Overlap50(t *testing.T) {
	cfg := newFOVConfig(35, 23.6, 15.8, 50)
	fov, _ := NewFOVCalculator(cfg)

	hFOV := fov.HorizontalFOV()
	want := hFOV * 0.5
	got := fov.HorizontalRotationAngle()
	if math.Abs(got-want) > epsilon {
		t.Errorf("with 50%% overlap, rotation (%v) should be FOV*0.5 (%v)", got, want)
	}
}

func TestFOVCalculator_DifferentFocalLengths(t *testing.T) {
	cases := []struct {
		name    string
		focal   float64
		wantGt0 bool
	}{
		{"wide_18mm", 18, true},
		{"normal_50mm", 50, true},
		{"tele_200mm", 200, true},
		{"tele_500mm", 500, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newFOVConfig(tc.focal, 23.6, 15.8, 30)
			fov, _ := NewFOVCalculator(cfg)
			if fov.HorizontalFOV() <= 0 {
				t.Error("HorizontalFOV should be > 0")
			}
			if fov.VerticalFOV() <= 0 {
				t.Error("VerticalFOV should be > 0")
			}
			if fov.HorizontalRotationAngle() <= 0 {
				t.Error("HorizontalRotationAngle should be > 0")
			}
		})
	}
}
