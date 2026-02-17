package main

import (
	"math"
	"testing"

	"github.com/cjeanneret/PanGo/internal/config"
	"github.com/cjeanneret/PanGo/internal/web"
)

// ---------- validateCLIOverrides ----------

func TestValidateCLIOverrides_AllZero(t *testing.T) {
	if err := validateCLIOverrides(0, 0, 0); err != nil {
		t.Errorf("all zeros should be valid (use config defaults), got: %v", err)
	}
}

func TestValidateCLIOverrides_ValidBoundary(t *testing.T) {
	cases := []struct {
		name string
		h, v, f float64
	}{
		{"min_horizontal", 1, 0, 0},
		{"max_horizontal", 360, 0, 0},
		{"min_vertical", 0, 1, 0},
		{"max_vertical", 0, 180, 0},
		{"min_focal", 0, 0, 1},
		{"max_focal", 0, 0, 500},
		{"all_min", 1, 1, 1},
		{"all_max", 360, 180, 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCLIOverrides(tc.h, tc.v, tc.f); err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
		})
	}
}

func TestValidateCLIOverrides_ValidMidRange(t *testing.T) {
	if err := validateCLIOverrides(180, 90, 35); err != nil {
		t.Errorf("mid-range values should be valid, got: %v", err)
	}
}

func TestValidateCLIOverrides_SmallPositive(t *testing.T) {
	if err := validateCLIOverrides(0.001, 0.001, 0.001); err != nil {
		t.Errorf("very small positive values should be valid, got: %v", err)
	}
}

func TestValidateCLIOverrides_OutOfRange(t *testing.T) {
	cases := []struct {
		name    string
		h, v, f float64
	}{
		{"horizontal_too_large", 361, 0, 0},
		{"vertical_too_large", 0, 181, 0},
		{"focal_too_large", 0, 0, 501},
		{"horizontal_negative", -1, 0, 0},
		{"vertical_negative", 0, -1, 0},
		{"focal_negative", 0, 0, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCLIOverrides(tc.h, tc.v, tc.f); err == nil {
				t.Error("expected error for out-of-range value, got nil")
			}
		})
	}
}

func TestValidateCLIOverrides_NaN(t *testing.T) {
	nan := math.NaN()
	cases := []struct {
		name    string
		h, v, f float64
	}{
		{"horizontal_NaN", nan, 0, 0},
		{"vertical_NaN", 0, nan, 0},
		{"focal_NaN", 0, 0, nan},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCLIOverrides(tc.h, tc.v, tc.f); err == nil {
				t.Error("expected error for NaN, got nil")
			}
		})
	}
}

func TestValidateCLIOverrides_Infinity(t *testing.T) {
	posInf := math.Inf(1)
	negInf := math.Inf(-1)
	cases := []struct {
		name    string
		h, v, f float64
	}{
		{"horizontal_+Inf", posInf, 0, 0},
		{"horizontal_-Inf", negInf, 0, 0},
		{"vertical_+Inf", 0, posInf, 0},
		{"vertical_-Inf", 0, negInf, 0},
		{"focal_+Inf", 0, 0, posInf},
		{"focal_-Inf", 0, 0, negInf},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCLIOverrides(tc.h, tc.v, tc.f); err == nil {
				t.Error("expected error for Infinity, got nil")
			}
		})
	}
}

// ---------- webPortFlag ----------

func TestWebPortFlag_EmptyString(t *testing.T) {
	w := &webPortFlag{defaultPort: 8080}
	if err := w.Set(""); err != nil {
		t.Fatalf("Set(\"\") error: %v", err)
	}
	if w.port() != 8080 {
		t.Errorf("expected default port 8080, got %d", w.port())
	}
}

func TestWebPortFlag_ValidPorts(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"8080", 8080},
		{"1", 1},
		{"65535", 65535},
		{"3000", 3000},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			w := &webPortFlag{defaultPort: 8080}
			if err := w.Set(tc.input); err != nil {
				t.Fatalf("Set(%q) error: %v", tc.input, err)
			}
			if w.port() != tc.want {
				t.Errorf("port() = %d, want %d", w.port(), tc.want)
			}
		})
	}
}

func TestWebPortFlag_InvalidPorts(t *testing.T) {
	cases := []string{"0", "65536", "-1", "abc", "8080.5"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			w := &webPortFlag{defaultPort: 8080}
			if err := w.Set(input); err == nil {
				t.Errorf("Set(%q) should fail, got nil", input)
			}
		})
	}
}

func TestWebPortFlag_String(t *testing.T) {
	w := &webPortFlag{val: 0}
	if s := w.String(); s != "0" {
		t.Errorf("String() = %q, want \"0\"", s)
	}
	w.val = 9090
	if s := w.String(); s != "9090" {
		t.Errorf("String() = %q, want \"9090\"", s)
	}
}

// ---------- applyOverrides ----------

func newTestConfig() *config.Config {
	return &config.Config{
		Camera: config.CameraConfig{Type: "nikon_d90_gpio"},
		Lens:   config.LensConfig{FocalLengthMm: 35.0},
		Sensor: &config.SensorConfig{WidthMm: 23.6, HeightMm: 15.8},
		PanStepper: config.StepperConfig{
			StepPin: 17, DirPin: 27, EnablePin: 5,
			StepsPerRev: 200, Microstepping: 16,
		},
		TiltStepper: config.StepperConfig{
			StepPin: 22, DirPin: 23, EnablePin: 6,
			StepsPerRev: 200, Microstepping: 16,
		},
		Defaults: config.DefaultsConfig{
			MoveSpeedMs:        2,
			OverlapPercent:     30.0,
			HorizontalAngleDeg: 180.0,
			VerticalAngleDeg:   30.0,
			MockGPIO:           true,
		},
	}
}

func TestApplyOverrides_NonZero(t *testing.T) {
	cfg := newTestConfig()
	applyOverrides(cfg, web.Overrides{
		HorizontalAngleDeg: 270.0,
		VerticalAngleDeg:   60.0,
		FocalLengthMm:      50.0,
	})
	if cfg.Defaults.HorizontalAngleDeg != 270.0 {
		t.Errorf("HorizontalAngleDeg = %v, want 270.0", cfg.Defaults.HorizontalAngleDeg)
	}
	if cfg.Defaults.VerticalAngleDeg != 60.0 {
		t.Errorf("VerticalAngleDeg = %v, want 60.0", cfg.Defaults.VerticalAngleDeg)
	}
	if cfg.Lens.FocalLengthMm != 50.0 {
		t.Errorf("FocalLengthMm = %v, want 50.0", cfg.Lens.FocalLengthMm)
	}
}

func TestApplyOverrides_ZeroLeavesUnchanged(t *testing.T) {
	cfg := newTestConfig()
	origH := cfg.Defaults.HorizontalAngleDeg
	origV := cfg.Defaults.VerticalAngleDeg
	origF := cfg.Lens.FocalLengthMm

	applyOverrides(cfg, web.Overrides{})

	if cfg.Defaults.HorizontalAngleDeg != origH {
		t.Errorf("HorizontalAngleDeg changed: %v != %v", cfg.Defaults.HorizontalAngleDeg, origH)
	}
	if cfg.Defaults.VerticalAngleDeg != origV {
		t.Errorf("VerticalAngleDeg changed: %v != %v", cfg.Defaults.VerticalAngleDeg, origV)
	}
	if cfg.Lens.FocalLengthMm != origF {
		t.Errorf("FocalLengthMm changed: %v != %v", cfg.Lens.FocalLengthMm, origF)
	}
}

func TestApplyOverrides_Partial(t *testing.T) {
	cfg := newTestConfig()
	origV := cfg.Defaults.VerticalAngleDeg
	origF := cfg.Lens.FocalLengthMm

	applyOverrides(cfg, web.Overrides{HorizontalAngleDeg: 300.0})

	if cfg.Defaults.HorizontalAngleDeg != 300.0 {
		t.Errorf("HorizontalAngleDeg = %v, want 300.0", cfg.Defaults.HorizontalAngleDeg)
	}
	if cfg.Defaults.VerticalAngleDeg != origV {
		t.Errorf("VerticalAngleDeg should be unchanged: %v != %v", cfg.Defaults.VerticalAngleDeg, origV)
	}
	if cfg.Lens.FocalLengthMm != origF {
		t.Errorf("FocalLengthMm should be unchanged: %v != %v", cfg.Lens.FocalLengthMm, origF)
	}
}

// ---------- applyOverridesToCopy ----------

func TestApplyOverridesToCopy_OriginalUnmutated(t *testing.T) {
	cfg := newTestConfig()
	origH := cfg.Defaults.HorizontalAngleDeg

	copy := applyOverridesToCopy(cfg, web.Overrides{HorizontalAngleDeg: 999.0})

	if cfg.Defaults.HorizontalAngleDeg != origH {
		t.Errorf("original mutated: HorizontalAngleDeg = %v, want %v", cfg.Defaults.HorizontalAngleDeg, origH)
	}
	if copy.Defaults.HorizontalAngleDeg != 999.0 {
		t.Errorf("copy HorizontalAngleDeg = %v, want 999.0", copy.Defaults.HorizontalAngleDeg)
	}
}

func TestApplyOverridesToCopy_ZeroOverrides(t *testing.T) {
	cfg := newTestConfig()
	copy := applyOverridesToCopy(cfg, web.Overrides{})

	if copy.Defaults.HorizontalAngleDeg != cfg.Defaults.HorizontalAngleDeg {
		t.Errorf("HorizontalAngleDeg mismatch")
	}
	if copy.Defaults.VerticalAngleDeg != cfg.Defaults.VerticalAngleDeg {
		t.Errorf("VerticalAngleDeg mismatch")
	}
	if copy.Lens.FocalLengthMm != cfg.Lens.FocalLengthMm {
		t.Errorf("FocalLengthMm mismatch")
	}
}

func TestApplyOverridesToCopy_PreservesNestedFields(t *testing.T) {
	cfg := newTestConfig()
	copy := applyOverridesToCopy(cfg, web.Overrides{HorizontalAngleDeg: 100.0})

	if copy.PanStepper.StepsPerRev != cfg.PanStepper.StepsPerRev {
		t.Errorf("PanStepper.StepsPerRev not preserved")
	}
	if copy.TiltStepper.Microstepping != cfg.TiltStepper.Microstepping {
		t.Errorf("TiltStepper.Microstepping not preserved")
	}
	if copy.Camera.Type != cfg.Camera.Type {
		t.Errorf("Camera.Type not preserved")
	}
	if copy.Defaults.OverlapPercent != cfg.Defaults.OverlapPercent {
		t.Errorf("OverlapPercent not preserved")
	}
}

func TestApplyOverridesToCopy_ReturnsNewPointer(t *testing.T) {
	cfg := newTestConfig()
	copy := applyOverridesToCopy(cfg, web.Overrides{})
	if copy == cfg {
		t.Error("applyOverridesToCopy should return a new pointer, got same address")
	}
}

// ---------- Cross-source consistency ----------

func TestOverrides_CLIAndWebProduceSameResult(t *testing.T) {
	overrides := web.Overrides{
		HorizontalAngleDeg: 270.0,
		VerticalAngleDeg:   45.0,
		FocalLengthMm:      50.0,
	}

	// Simulate CLI path: mutate config directly
	cfgCLI := newTestConfig()
	applyOverrides(cfgCLI, overrides)

	// Simulate web path: copy then override
	cfgWeb := newTestConfig()
	cfgWebCopy := applyOverridesToCopy(cfgWeb, overrides)

	if cfgCLI.Defaults.HorizontalAngleDeg != cfgWebCopy.Defaults.HorizontalAngleDeg {
		t.Errorf("HorizontalAngleDeg differs: CLI=%v, Web=%v",
			cfgCLI.Defaults.HorizontalAngleDeg, cfgWebCopy.Defaults.HorizontalAngleDeg)
	}
	if cfgCLI.Defaults.VerticalAngleDeg != cfgWebCopy.Defaults.VerticalAngleDeg {
		t.Errorf("VerticalAngleDeg differs: CLI=%v, Web=%v",
			cfgCLI.Defaults.VerticalAngleDeg, cfgWebCopy.Defaults.VerticalAngleDeg)
	}
	if cfgCLI.Lens.FocalLengthMm != cfgWebCopy.Lens.FocalLengthMm {
		t.Errorf("FocalLengthMm differs: CLI=%v, Web=%v",
			cfgCLI.Lens.FocalLengthMm, cfgWebCopy.Lens.FocalLengthMm)
	}
}
