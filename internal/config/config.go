package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// StepperConfig holds the configuration for a stepper motor.
type StepperConfig struct {
	StepPin       int `yaml:"step_pin"`
	DirPin        int `yaml:"dir_pin"`
	EnablePin     int `yaml:"enable_pin"` // A4988 ENABLE pin (BCM). 0 = not used. Active LOW.
	StepsPerRev   int `yaml:"steps_per_rev"`
	Microstepping int `yaml:"microstepping"`
}

// CameraConfig describes how to communicate with the camera.
// Type selects a concrete implementation (e.g., "nikon_d90_gpio").
type CameraConfig struct {
	Type            string `yaml:"type"`               // e.g., "nikon_d90_gpio"
	FocusPin        int    `yaml:"focus_pin"`          // GPIO pin for FOCUS line
	ShutterPin      int    `yaml:"shutter_pin"`        // GPIO pin for SHUTTER line
	FocusDelayMs    int    `yaml:"focus_delay_ms"`     // autofocus delay (ms)
	ShutterDelayMs  int    `yaml:"shutter_delay_ms"`   // shutter hold time (ms)
	PostShotDelayMs int    `yaml:"post_shot_delay_ms"` // delay after shot before movement (ms)
	// Note: GND is physically connected to Raspberry Pi ground
}

// LensConfig describes the mounted lens.
type LensConfig struct {
	Name          string  `yaml:"name"`            // e.g., "Nikkor 35mm f/1.8"
	FocalLengthMm float64 `yaml:"focal_length_mm"` // focal length in use (or main focal length for zoom)
}

// SensorConfig is optional: physical sensor size in mm.
type SensorConfig struct {
	WidthMm  float64 `yaml:"width_mm"`  // e.g., 23.6 for Nikon APS-C
	HeightMm float64 `yaml:"height_mm"` // e.g., 15.8
}

// ResolutionConfig is optional: sensor/image resolution in pixels.
type ResolutionConfig struct {
	WidthPx  int `yaml:"width_px"`  // e.g., 4288
	HeightPx int `yaml:"height_px"` // e.g., 2848
}

// DefaultsConfig contains generic parameters (speed, etc.).
type DefaultsConfig struct {
	MoveSpeedMs        int     `yaml:"move_speed_ms"`        // delay between motor steps
	OverlapPercent     float64 `yaml:"overlap_percent"`      // desired overlap between photos (0-100)
	HorizontalAngleDeg float64 `yaml:"horizontal_angle_deg"` // total horizontal shooting angle (default: 180°)
	VerticalAngleDeg   float64 `yaml:"vertical_angle_deg"`   // total vertical shooting angle (default: 30°)
	DebugLevel         int     `yaml:"debug_level"`          // debug level 0-4 (0=off, 1=info, 2=live, 3=verbose, 4=trace)
	MockGPIO           bool    `yaml:"mock_gpio"`            // use mock GPIO (true=dev/test, false=real Raspberry Pi)
}

// MaxConfigFileBytes is the maximum allowed size for a config file (256 KB).
// Prevents memory exhaustion on resource-constrained devices (e.g. Raspberry Pi Zero 2).
const MaxConfigFileBytes = 256 << 10

// Config aggregates all application configuration.
type Config struct {
	PanStepper  StepperConfig     `yaml:"pan_stepper"`
	TiltStepper StepperConfig     `yaml:"tilt_stepper"`
	Camera      CameraConfig      `yaml:"camera"`
	Lens        LensConfig        `yaml:"lens"`
	Sensor      *SensorConfig     `yaml:"sensor,omitempty"`     // optional
	Resolution  *ResolutionConfig `yaml:"resolution,omitempty"` // optional
	Defaults    DefaultsConfig    `yaml:"defaults"`
}

const (
	MinGPIOPin           = 0
	MaxGPIOPin           = 27
	MaxMotorStepsPerRev  = 1000
	MaxCameraDelayMs     = 60000
	MaxFocalLengthMm     = 2000.0
	MinFocalLengthMm     = 1.0
	MaxSensorDimensionMm = 100.0
)

var validMicrostepping = map[int]bool{
	1: true, 2: true, 4: true, 8: true, 16: true, 32: true,
}

func validateGPIOPin(pin int, name string) error {
	if pin < MinGPIOPin || pin > MaxGPIOPin {
		return fmt.Errorf("%s must be between %d and %d (BCM pin number), got %d", name, MinGPIOPin, MaxGPIOPin, pin)
	}
	return nil
}

func validateStepperConfig(cfg StepperConfig, name string) error {
	if err := validateGPIOPin(cfg.StepPin, name+" step_pin"); err != nil {
		return err
	}
	if err := validateGPIOPin(cfg.DirPin, name+" dir_pin"); err != nil {
		return err
	}
	if cfg.EnablePin != 0 {
		if err := validateGPIOPin(cfg.EnablePin, name+" enable_pin"); err != nil {
			return err
		}
	}
	if cfg.StepsPerRev <= 0 {
		return fmt.Errorf("%s steps_per_rev must be > 0, got %d", name, cfg.StepsPerRev)
	}
	if cfg.StepsPerRev > MaxMotorStepsPerRev {
		return fmt.Errorf("%s steps_per_rev must be <= %d, got %d", name, MaxMotorStepsPerRev, cfg.StepsPerRev)
	}
	if !validMicrostepping[cfg.Microstepping] {
		return fmt.Errorf("%s microstepping must be one of 1,2,4,8,16,32, got %d", name, cfg.Microstepping)
	}
	return nil
}

func validateCameraConfig(cfg CameraConfig) error {
	if err := validateGPIOPin(cfg.FocusPin, "camera focus_pin"); err != nil {
		return err
	}
	if err := validateGPIOPin(cfg.ShutterPin, "camera shutter_pin"); err != nil {
		return err
	}
	if cfg.FocusDelayMs < 0 || cfg.FocusDelayMs > MaxCameraDelayMs {
		return fmt.Errorf("camera focus_delay_ms must be between 0 and %d ms, got %d", MaxCameraDelayMs, cfg.FocusDelayMs)
	}
	if cfg.ShutterDelayMs < 0 || cfg.ShutterDelayMs > MaxCameraDelayMs {
		return fmt.Errorf("camera shutter_delay_ms must be between 0 and %d ms, got %d", MaxCameraDelayMs, cfg.ShutterDelayMs)
	}
	if cfg.PostShotDelayMs < 0 || cfg.PostShotDelayMs > MaxCameraDelayMs {
		return fmt.Errorf("camera post_shot_delay_ms must be between 0 and %d ms, got %d", MaxCameraDelayMs, cfg.PostShotDelayMs)
	}
	return nil
}

func validateLensConfig(cfg LensConfig) error {
	if cfg.FocalLengthMm < MinFocalLengthMm || cfg.FocalLengthMm > MaxFocalLengthMm {
		return fmt.Errorf("lens focal_length_mm must be between %.0f and %.0f mm, got %.2f", MinFocalLengthMm, MaxFocalLengthMm, cfg.FocalLengthMm)
	}
	return nil
}

func validateSensorConfig(cfg *SensorConfig) error {
	if cfg.WidthMm <= 0 || cfg.WidthMm > MaxSensorDimensionMm {
		return fmt.Errorf("sensor width_mm must be between 0 and %.0f mm, got %.2f", MaxSensorDimensionMm, cfg.WidthMm)
	}
	if cfg.HeightMm <= 0 || cfg.HeightMm > MaxSensorDimensionMm {
		return fmt.Errorf("sensor height_mm must be between 0 and %.0f mm, got %.2f", MaxSensorDimensionMm, cfg.HeightMm)
	}
	return nil
}

// ValidateConfigPath ensures the path is within a configs/ directory and has .yaml extension.
// Prevents path traversal (e.g. ../../etc/passwd) when loading configuration.
func ValidateConfigPath(path string) error {
	cleaned := filepath.Clean(path)
	if filepath.Ext(cleaned) != ".yaml" {
		return fmt.Errorf("config file must have .yaml extension, got %s", filepath.Ext(cleaned))
	}
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	// Ensure the config file lives in a directory named "configs" (works regardless of CWD).
	parentDir := filepath.Dir(absPath)
	if filepath.Base(parentDir) != "configs" {
		return fmt.Errorf("config path must be within a directory named configs/")
	}
	return nil
}

// Load reads a YAML file and returns the configuration.
func Load(path string) (*Config, error) {
	if err := ValidateConfigPath(path); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	if info.Size() > MaxConfigFileBytes {
		return nil, fmt.Errorf("config file too large: %d bytes (max %d)", info.Size(), MaxConfigFileBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	// Validate stepper configurations
	if err := validateStepperConfig(cfg.PanStepper, "pan_stepper"); err != nil {
		return nil, err
	}
	if err := validateStepperConfig(cfg.TiltStepper, "tilt_stepper"); err != nil {
		return nil, err
	}

	// Validate camera configuration
	if cfg.Camera.Type == "" {
		return nil, fmt.Errorf("camera.type is required")
	}
	if err := validateCameraConfig(cfg.Camera); err != nil {
		return nil, err
	}

	// Validate lens configuration
	if err := validateLensConfig(cfg.Lens); err != nil {
		return nil, err
	}

	// Validate sensor configuration if provided
	if cfg.Sensor != nil {
		if err := validateSensorConfig(cfg.Sensor); err != nil {
			return nil, err
		}
	}
	const MaxMoveSpeedMs = 1000
	if cfg.Defaults.MoveSpeedMs <= 0 {
		cfg.Defaults.MoveSpeedMs = 2 // reasonable default
	}
	if cfg.Defaults.MoveSpeedMs > MaxMoveSpeedMs {
		cfg.Defaults.MoveSpeedMs = MaxMoveSpeedMs // cap at max
	}
	if cfg.Defaults.OverlapPercent < 0 || cfg.Defaults.OverlapPercent > 100 {
		return nil, fmt.Errorf("overlap_percent must be between 0 and 100, got %.2f", cfg.Defaults.OverlapPercent)
	}
	if cfg.Defaults.OverlapPercent == 0 {
		cfg.Defaults.OverlapPercent = 30 // reasonable default (30%)
	}
	if cfg.Defaults.HorizontalAngleDeg <= 0 {
		cfg.Defaults.HorizontalAngleDeg = 180 // default (180°)
	}
	if cfg.Defaults.VerticalAngleDeg <= 0 {
		cfg.Defaults.VerticalAngleDeg = 30 // default (30°)
	}
	if cfg.Defaults.HorizontalAngleDeg > 360 {
		return nil, fmt.Errorf("horizontal_angle_deg must be <= 360, got %.2f", cfg.Defaults.HorizontalAngleDeg)
	}
	if cfg.Defaults.VerticalAngleDeg > 180 {
		return nil, fmt.Errorf("vertical_angle_deg must be <= 180, got %.2f", cfg.Defaults.VerticalAngleDeg)
	}

	// Default values for camera delays (if not set)
	if cfg.Camera.FocusDelayMs == 0 {
		cfg.Camera.FocusDelayMs = 500 // 500ms for autofocus
	}
	if cfg.Camera.ShutterDelayMs == 0 {
		cfg.Camera.ShutterDelayMs = 200 // 200ms shutter hold
	}
	if cfg.Camera.PostShotDelayMs == 0 {
		cfg.Camera.PostShotDelayMs = 300 // 300ms after shot before movement
	}

	// Validate debug level is in valid range
	if cfg.Defaults.DebugLevel < 0 || cfg.Defaults.DebugLevel > 4 {
		return nil, fmt.Errorf("debug_level must be between 0 and 4, got %d", cfg.Defaults.DebugLevel)
	}

	return &cfg, nil
}

// MoveSpeed returns the duration between two motor steps.
func (c *Config) MoveSpeed() time.Duration {
	return time.Duration(c.Defaults.MoveSpeedMs) * time.Millisecond
}

// OverlapRatio returns the overlap as a ratio (0.0 to 1.0).
// For example, 30% becomes 0.3.
func (c *Config) OverlapRatio() float64 {
	return c.Defaults.OverlapPercent / 100.0
}

// OverlapPercent returns the overlap in percent (0.0 to 100.0).
func (c *Config) OverlapPercent() float64 {
	return c.Defaults.OverlapPercent
}

// HorizontalAngleDeg returns the total horizontal shooting angle in degrees.
func (c *Config) HorizontalAngleDeg() float64 {
	return c.Defaults.HorizontalAngleDeg
}

// VerticalAngleDeg returns the total vertical shooting angle in degrees.
func (c *Config) VerticalAngleDeg() float64 {
	return c.Defaults.VerticalAngleDeg
}

// HorizontalHalfAngleDeg returns half of the horizontal angle.
// Useful for calculating displacement from center (left/right).
func (c *Config) HorizontalHalfAngleDeg() float64 {
	return c.Defaults.HorizontalAngleDeg / 2.0
}

// VerticalHalfAngleDeg returns half of the vertical angle.
// Useful for calculating displacement from center (up/down).
func (c *Config) VerticalHalfAngleDeg() float64 {
	return c.Defaults.VerticalAngleDeg / 2.0
}

// FocusDelay returns the autofocus delay duration.
func (c *Config) FocusDelay() time.Duration {
	return time.Duration(c.Camera.FocusDelayMs) * time.Millisecond
}

// ShutterDelay returns the shutter hold duration.
func (c *Config) ShutterDelay() time.Duration {
	return time.Duration(c.Camera.ShutterDelayMs) * time.Millisecond
}

// PostShotDelay returns the delay after shot before movement.
func (c *Config) PostShotDelay() time.Duration {
	return time.Duration(c.Camera.PostShotDelayMs) * time.Millisecond
}
