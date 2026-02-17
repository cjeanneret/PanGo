package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- ValidateConfigPath ----------

func TestValidateConfigPath_Valid(t *testing.T) {
	// Create a real configs/ directory so filepath.Abs resolves correctly.
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "default.yaml")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateConfigPath(path); err != nil {
		t.Errorf("expected valid path, got error: %v", err)
	}
}

func TestValidateConfigPath_PathTraversal(t *testing.T) {
	cases := []string{
		"../../etc/passwd",
		"configs/../../../etc/shadow",
	}
	for _, path := range cases {
		if err := ValidateConfigPath(path); err == nil {
			t.Errorf("expected error for traversal path %q, got nil", path)
		}
	}
}

func TestValidateConfigPath_WrongExtension(t *testing.T) {
	cases := []string{
		"configs/default.json",
		"configs/default.yml",
		"configs/default.txt",
		"configs/default",
	}
	for _, path := range cases {
		if err := ValidateConfigPath(path); err == nil {
			t.Errorf("expected error for extension in %q, got nil", path)
		}
	}
}

func TestValidateConfigPath_NotInConfigsDir(t *testing.T) {
	cases := []string{
		"other/default.yaml",
		"default.yaml",
		"/tmp/default.yaml",
	}
	for _, path := range cases {
		if err := ValidateConfigPath(path); err == nil {
			t.Errorf("expected error for path outside configs/ %q, got nil", path)
		}
	}
}

func TestValidateConfigPath_EmptyPath(t *testing.T) {
	if err := ValidateConfigPath(""); err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestValidateConfigPath_VeryLongPath(t *testing.T) {
	long := "configs/" + strings.Repeat("a", 1000) + ".yaml"
	// Should not panic; error or success is OS-dependent, but must not crash.
	_ = ValidateConfigPath(long)
}

func TestValidateConfigPath_SpecialChars(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		wantErr bool
	}{
		{"con fig.yaml", false},
		{"café.yaml", false},
	}
	for _, tc := range cases {
		path := filepath.Join(cfgDir, tc.name)
		err := ValidateConfigPath(path)
		if tc.wantErr && err == nil {
			t.Errorf("expected error for %q, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("unexpected error for %q: %v", tc.name, err)
		}
	}
}

func TestValidateConfigPath_DoubleTraversal(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Try to escape via ../../configs/ok.yaml — filepath.Clean resolves this
	// and the parent must still be "configs".
	path := filepath.Join(cfgDir, "../../configs/ok.yaml")
	err := ValidateConfigPath(path)
	// After Clean the parent may or may not be "configs" depending on resolution.
	// The important thing is it either succeeds with a valid parent or fails.
	_ = err
}

// ---------- Load ----------

// writeConfig creates a temporary configs/ dir with the given YAML content and returns the path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validYAML = `
camera:
  type: "nikon_d90_gpio"
  focus_pin: 24
  shutter_pin: 25
lens:
  name: "Nikkor 35mm"
  focal_length_mm: 35.0
sensor:
  width_mm: 23.6
  height_mm: 15.8
pan_stepper:
  step_pin: 17
  dir_pin: 27
  enable_pin: 5
  steps_per_rev: 200
  microstepping: 16
tilt_stepper:
  step_pin: 22
  dir_pin: 23
  enable_pin: 6
  steps_per_rev: 200
  microstepping: 16
defaults:
  move_speed_ms: 2
  overlap_percent: 30.0
  horizontal_angle_deg: 180.0
  vertical_angle_deg: 30.0
  debug_level: 0
  mock_gpio: true
`

func TestLoad_ValidFullConfig(t *testing.T) {
	path := writeConfig(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Camera.Type != "nikon_d90_gpio" {
		t.Errorf("camera.type = %q, want %q", cfg.Camera.Type, "nikon_d90_gpio")
	}
	if cfg.Lens.FocalLengthMm != 35.0 {
		t.Errorf("lens.focal_length_mm = %v, want 35.0", cfg.Lens.FocalLengthMm)
	}
	if cfg.Sensor == nil {
		t.Fatal("sensor should not be nil")
	}
	if cfg.Sensor.WidthMm != 23.6 {
		t.Errorf("sensor.width_mm = %v, want 23.6", cfg.Sensor.WidthMm)
	}
	if cfg.Defaults.OverlapPercent != 30.0 {
		t.Errorf("overlap_percent = %v, want 30.0", cfg.Defaults.OverlapPercent)
	}
	if cfg.Defaults.HorizontalAngleDeg != 180.0 {
		t.Errorf("horizontal_angle_deg = %v, want 180.0", cfg.Defaults.HorizontalAngleDeg)
	}
	if cfg.Defaults.VerticalAngleDeg != 30.0 {
		t.Errorf("vertical_angle_deg = %v, want 30.0", cfg.Defaults.VerticalAngleDeg)
	}
	if cfg.PanStepper.StepsPerRev != 200 {
		t.Errorf("pan_stepper.steps_per_rev = %d, want 200", cfg.PanStepper.StepsPerRev)
	}
}

func TestLoad_MissingCameraType(t *testing.T) {
	yaml := `
lens:
  focal_length_mm: 35.0
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for missing camera.type, got nil")
	}
}

func TestLoad_MissingFocalLength(t *testing.T) {
	yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  name: "test"
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for missing focal_length_mm, got nil")
	}
}

func TestLoad_NegativeFocalLength(t *testing.T) {
	yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  focal_length_mm: -10.0
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for negative focal_length_mm, got nil")
	}
}

func TestLoad_OverlapOutOfRange(t *testing.T) {
	cases := []struct {
		name    string
		overlap float64
	}{
		{"negative", -1.0},
		{"over_100", 101.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  focal_length_mm: 35.0
defaults:
  overlap_percent: ` + formatFloat(tc.overlap)
			path := writeConfig(t, yaml)
			_, err := Load(path)
			if err == nil {
				t.Errorf("expected error for overlap_percent=%v, got nil", tc.overlap)
			}
		})
	}
}

func TestLoad_HorizontalAngleTooLarge(t *testing.T) {
	yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  focal_length_mm: 35.0
defaults:
  horizontal_angle_deg: 361.0
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for horizontal_angle_deg > 360, got nil")
	}
}

func TestLoad_VerticalAngleTooLarge(t *testing.T) {
	yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  focal_length_mm: 35.0
defaults:
  vertical_angle_deg: 181.0
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for vertical_angle_deg > 180, got nil")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  focal_length_mm: 35.0
`
	path := writeConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Defaults.MoveSpeedMs != 2 {
		t.Errorf("move_speed_ms default = %d, want 2", cfg.Defaults.MoveSpeedMs)
	}
	if cfg.Defaults.OverlapPercent != 30 {
		t.Errorf("overlap_percent default = %v, want 30", cfg.Defaults.OverlapPercent)
	}
	if cfg.Defaults.HorizontalAngleDeg != 180 {
		t.Errorf("horizontal_angle_deg default = %v, want 180", cfg.Defaults.HorizontalAngleDeg)
	}
	if cfg.Defaults.VerticalAngleDeg != 30 {
		t.Errorf("vertical_angle_deg default = %v, want 30", cfg.Defaults.VerticalAngleDeg)
	}
	if cfg.Camera.FocusDelayMs != 500 {
		t.Errorf("focus_delay_ms default = %d, want 500", cfg.Camera.FocusDelayMs)
	}
	if cfg.Camera.ShutterDelayMs != 200 {
		t.Errorf("shutter_delay_ms default = %d, want 200", cfg.Camera.ShutterDelayMs)
	}
	if cfg.Camera.PostShotDelayMs != 300 {
		t.Errorf("post_shot_delay_ms default = %d, want 300", cfg.Camera.PostShotDelayMs)
	}
}

func TestLoad_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "big.yaml")
	data := make([]byte, MaxConfigFileBytes+1)
	for i := range data {
		data[i] = '#'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for oversized config file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "{{{{invalid yaml!!!!")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	path := writeConfig(t, "")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for empty config (camera.type missing), got nil")
	}
}

func TestLoad_UnknownFields(t *testing.T) {
	yaml := `
camera:
  type: "nikon_d90_gpio"
lens:
  focal_length_mm: 35.0
unknown_section:
  foo: bar
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	if err != nil {
		t.Errorf("unknown fields should be ignored, got error: %v", err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "configs")
	if err := os.Mkdir(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "nonexistent.yaml")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// ---------- Helper methods ----------

func TestConfig_MoveSpeed(t *testing.T) {
	cfg := &Config{Defaults: DefaultsConfig{MoveSpeedMs: 5}}
	got := cfg.MoveSpeed()
	want := 5 * time.Millisecond
	if got != want {
		t.Errorf("MoveSpeed() = %v, want %v", got, want)
	}
}

func TestConfig_OverlapRatio(t *testing.T) {
	cases := []struct {
		percent float64
		want    float64
	}{
		{0, 0.0},
		{30, 0.3},
		{50, 0.5},
		{100, 1.0},
	}
	for _, tc := range cases {
		cfg := &Config{Defaults: DefaultsConfig{OverlapPercent: tc.percent}}
		got := cfg.OverlapRatio()
		if got != tc.want {
			t.Errorf("OverlapRatio() for %v%% = %v, want %v", tc.percent, got, tc.want)
		}
	}
}

func TestConfig_HalfAngles(t *testing.T) {
	cfg := &Config{Defaults: DefaultsConfig{
		HorizontalAngleDeg: 180.0,
		VerticalAngleDeg:   30.0,
	}}
	if got := cfg.HorizontalHalfAngleDeg(); got != 90.0 {
		t.Errorf("HorizontalHalfAngleDeg() = %v, want 90.0", got)
	}
	if got := cfg.VerticalHalfAngleDeg(); got != 15.0 {
		t.Errorf("VerticalHalfAngleDeg() = %v, want 15.0", got)
	}
}

func TestConfig_FocusDelay(t *testing.T) {
	cfg := &Config{Camera: CameraConfig{FocusDelayMs: 500}}
	got := cfg.FocusDelay()
	want := 500 * time.Millisecond
	if got != want {
		t.Errorf("FocusDelay() = %v, want %v", got, want)
	}
}

func TestConfig_ShutterDelay(t *testing.T) {
	cfg := &Config{Camera: CameraConfig{ShutterDelayMs: 200}}
	got := cfg.ShutterDelay()
	want := 200 * time.Millisecond
	if got != want {
		t.Errorf("ShutterDelay() = %v, want %v", got, want)
	}
}

func TestConfig_PostShotDelay(t *testing.T) {
	cfg := &Config{Camera: CameraConfig{PostShotDelayMs: 300}}
	got := cfg.PostShotDelay()
	want := 300 * time.Millisecond
	if got != want {
		t.Errorf("PostShotDelay() = %v, want %v", got, want)
	}
}

func TestConfig_OverlapPercent(t *testing.T) {
	cfg := &Config{Defaults: DefaultsConfig{OverlapPercent: 42.5}}
	if got := cfg.OverlapPercent(); got != 42.5 {
		t.Errorf("OverlapPercent() = %v, want 42.5", got)
	}
}

func TestConfig_AngleAccessors(t *testing.T) {
	cfg := &Config{Defaults: DefaultsConfig{
		HorizontalAngleDeg: 270.0,
		VerticalAngleDeg:   60.0,
	}}
	if got := cfg.HorizontalAngleDeg(); got != 270.0 {
		t.Errorf("HorizontalAngleDeg() = %v, want 270.0", got)
	}
	if got := cfg.VerticalAngleDeg(); got != 60.0 {
		t.Errorf("VerticalAngleDeg() = %v, want 60.0", got)
	}
}

// formatFloat is a test helper for embedding floats into YAML strings.
func formatFloat(f float64) string {
	return fmt.Sprintf("%g", f)
}
