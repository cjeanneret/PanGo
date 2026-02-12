package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/cjeanneret/PanGo/internal/config"
	"github.com/cjeanneret/PanGo/internal/debug"
	"github.com/cjeanneret/PanGo/internal/hw/camera"
	"github.com/cjeanneret/PanGo/internal/hw/gpio"
	"github.com/cjeanneret/PanGo/internal/hw/stepper"
	"github.com/cjeanneret/PanGo/internal/logic/capture"
	"github.com/cjeanneret/PanGo/internal/logic/geometry"
	"github.com/cjeanneret/PanGo/internal/logic/motion"
	"github.com/cjeanneret/PanGo/internal/web"
)

func main() {
	// CLI flags
	webPort := &webPortFlag{defaultPort: 8080}
	flag.Var(webPort, "web", "start web server on port; -web= for default 8080, -web 8980 for custom port")
	cfgPath := flag.String("config", filepath.Join("configs", "default.yaml"), "path to config file")
	horizontalAngleDeg := flag.Float64("horizontal_angle_deg", 0, "override horizontal angle in degrees (1-360)")
	verticalAngleDeg := flag.Float64("vertical_angle_deg", 0, "override vertical angle in degrees (1-180)")
	focalLengthMm := flag.Float64("focal_length_mm", 0, "override focal length in mm")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	// Validate CLI overrides (only non-zero values are applied; zero means "use config default")
	if err := validateCLIOverrides(*horizontalAngleDeg, *verticalAngleDeg, *focalLengthMm); err != nil {
		log.Fatalf("invalid CLI override: %v", err)
	}

	// Apply CLI overrides to config
	applyOverrides(cfg, web.Overrides{
		HorizontalAngleDeg: *horizontalAngleDeg,
		VerticalAngleDeg:   *verticalAngleDeg,
		FocalLengthMm:     *focalLengthMm,
	})

	// Initialize debug system
	debug.Init(cfg.Defaults.DebugLevel)
	debug.Section("Initialization")
	debug.Value("Config path", *cfgPath)
	debug.Value("Debug level", cfg.Defaults.DebugLevel)

	// Initialize GPIO driver
	debug.Value("Mock GPIO", cfg.Defaults.MockGPIO)
	debug.Step(1, "Initializing GPIO driver")
	gpioDriver, err := gpio.NewDriver(cfg.Defaults.MockGPIO)
	if err != nil {
		log.Fatalf("init GPIO failed: %v", err)
	}
	defer func() {
		if err := gpioDriver.Close(); err != nil {
			log.Printf("closing GPIO driver failed: %v", err)
		}
	}()

	// Initialize stepper motors
	debug.Step(2, "Initializing stepper motors")
	stepDelay := cfg.MoveSpeed() / 2
	panMotor := stepper.NewStepper(gpioDriver, stepper.Config{
		StepPin:       cfg.PanStepper.StepPin,
		DirPin:        cfg.PanStepper.DirPin,
		EnablePin:     cfg.PanStepper.EnablePin,
		StepsPerRev:   cfg.PanStepper.StepsPerRev,
		Microstepping: cfg.PanStepper.Microstepping,
		StepDelay:     stepDelay,
	})
	debug.PrintStruct("Pan stepper config", cfg.PanStepper)
	tiltMotor := stepper.NewStepper(gpioDriver, stepper.Config{
		StepPin:       cfg.TiltStepper.StepPin,
		DirPin:        cfg.TiltStepper.DirPin,
		EnablePin:     cfg.TiltStepper.EnablePin,
		StepsPerRev:   cfg.TiltStepper.StepsPerRev,
		Microstepping: cfg.TiltStepper.Microstepping,
		StepDelay:     stepDelay,
	})
	debug.PrintStruct("Tilt stepper config", cfg.TiltStepper)

	// Initialize camera
	debug.Step(3, "Initializing camera")
	cam, err := newCameraFromConfig(gpioDriver, cfg)
	if err != nil {
		log.Fatalf("init camera failed: %v", err)
	}
	debug.Value("Camera type", cfg.Camera.Type)
	debug.Value("Focus pin", cfg.Camera.FocusPin)
	debug.Value("Shutter pin", cfg.Camera.ShutterPin)

	// Build runCapture closure over hardware and base config
	runCapture := func(ctx context.Context, overrides web.Overrides) error {
		return executeCapture(ctx, cfg, panMotor, tiltMotor, cam, overrides)
	}

	if port := webPort.port(); port > 0 {
		webAddr := fmt.Sprintf(":%d", port)
		broadcaster := web.NewStatusBroadcaster()
		debug.SetOutput(io.MultiWriter(os.Stdout, web.BroadcastWriter(broadcaster)))

		formDefaults := web.FormConfig{
			HorizontalAngleDeg: cfg.Defaults.HorizontalAngleDeg,
			VerticalAngleDeg:   cfg.Defaults.VerticalAngleDeg,
			FocalLengthMm:      cfg.Lens.FocalLengthMm,
		}
		srv := web.NewServer(webAddr, broadcaster, runCapture, formDefaults)
		if err := srv.Run(ctx); err != nil {
			log.Fatalf("web server: %v", err)
		}
		return
	}

	{
		// Run capture once with current config (already has CLI overrides applied)
		if err := runCapture(ctx, web.Overrides{}); err != nil {
			log.Fatalf("capture failed: %v", err)
		}
	}
}

// executeCapture runs the grid shot sequence with the given config and overrides.
// It applies overrides to a copy of the config, then runs the capture.
func executeCapture(
	ctx context.Context,
	baseCfg *config.Config,
	panMotor *stepper.Stepper,
	tiltMotor *stepper.Stepper,
	cam camera.Camera,
	overrides web.Overrides,
) error {
	cfg := applyOverridesToCopy(baseCfg, overrides)

	debug.Step(4, "Calculating grid plan")
	fovCalc, err := geometry.NewFOVCalculator(cfg)
	if err != nil {
		return fmt.Errorf("create FOV calculator: %w", err)
	}
	stepsCalc := geometry.NewStepsCalculator(cfg)
	gridPlan := geometry.CalculateGridPlan(cfg, fovCalc, stepsCalc)

	totalPhotos := gridPlan.PanColumns * gridPlan.TiltRows
	debug.Summary("Grid Plan Summary")
	debug.Grid(gridPlan.PanColumns, gridPlan.TiltRows, totalPhotos)
	debug.Info("Step sizes: pan=%d steps, tilt=%d steps", gridPlan.PanStepSize, gridPlan.TiltStepSize)

	debug.Section("Grid Plan Details")
	debug.Value("Pan columns", gridPlan.PanColumns)
	debug.Value("Tilt rows", gridPlan.TiltRows)
	debug.Value("Total photos", totalPhotos)
	debug.Value("Pan step size", gridPlan.PanStepSize)
	debug.Value("Tilt step size", gridPlan.TiltStepSize)
	debug.Value("Start pan steps", gridPlan.StartPanSteps)
	debug.Value("Start tilt steps", gridPlan.StartTiltSteps)
	debug.Value("Horizontal FOV", fovCalc.HorizontalFOV())
	debug.Value("Vertical FOV", fovCalc.VerticalFOV())
	debug.Value("Horizontal rotation angle", fovCalc.HorizontalRotationAngle())
	debug.Value("Vertical rotation angle", fovCalc.VerticalRotationAngle())

	debug.Step(5, "Creating motion and capture controllers")
	motionCtrl := motion.NewController(panMotor, tiltMotor)
	captureSeq := capture.NewSequence(motionCtrl, cam)

	debug.Section("Starting Grid Shot Sequence")
	err = captureSeq.RunGridShot(ctx, capture.GridShotParams{
		GridPlan:      gridPlan,
		Delay:         500 * time.Millisecond,
		MoveSpeed:     cfg.MoveSpeed(),
		ShotDelay:     300 * time.Millisecond,
		PostShotDelay: cfg.PostShotDelay(),
	})
	if err != nil {
		return err
	}

	debug.Section("Sequence Complete")
	return nil
}

// validateCLIOverrides checks that non-zero CLI overrides are within valid ranges.
// Zero values are ignored (they mean "use config default").
func validateCLIOverrides(horizontal, vertical, focal float64) error {
	if horizontal != 0 {
		if math.IsNaN(horizontal) || math.IsInf(horizontal, 0) || horizontal <= 0 || horizontal > 360 {
			return fmt.Errorf("horizontal_angle_deg must be between 1 and 360, got %g", horizontal)
		}
	}
	if vertical != 0 {
		if math.IsNaN(vertical) || math.IsInf(vertical, 0) || vertical <= 0 || vertical > 180 {
			return fmt.Errorf("vertical_angle_deg must be between 1 and 180, got %g", vertical)
		}
	}
	if focal != 0 {
		if math.IsNaN(focal) || math.IsInf(focal, 0) || focal <= 0 || focal > 500 {
			return fmt.Errorf("focal_length_mm must be between 1 and 500, got %g", focal)
		}
	}
	return nil
}

// applyOverrides mutates cfg with overrides. Only non-zero override values are applied.
func applyOverrides(cfg *config.Config, overrides web.Overrides) {
	if overrides.HorizontalAngleDeg > 0 {
		cfg.Defaults.HorizontalAngleDeg = overrides.HorizontalAngleDeg
	}
	if overrides.VerticalAngleDeg > 0 {
		cfg.Defaults.VerticalAngleDeg = overrides.VerticalAngleDeg
	}
	if overrides.FocalLengthMm > 0 {
		cfg.Lens.FocalLengthMm = overrides.FocalLengthMm
	}
}

// applyOverridesToCopy returns a new config with overrides applied.
// Zero values in overrides mean "use base config".
func applyOverridesToCopy(baseCfg *config.Config, overrides web.Overrides) *config.Config {
	cfg := *baseCfg
	if overrides.HorizontalAngleDeg > 0 {
		cfg.Defaults.HorizontalAngleDeg = overrides.HorizontalAngleDeg
	}
	if overrides.VerticalAngleDeg > 0 {
		cfg.Defaults.VerticalAngleDeg = overrides.VerticalAngleDeg
	}
	if overrides.FocalLengthMm > 0 {
		cfg.Lens.FocalLengthMm = overrides.FocalLengthMm
	}
	return &cfg
}

// webPortFlag implements flag.Value for -web: 0 = disabled, -web= or -web 8080 → 8080, -web 8980 → 8980.
type webPortFlag struct {
	val         int
	defaultPort int
}

func (w *webPortFlag) String() string {
	if w.val == 0 {
		return "0"
	}
	return strconv.Itoa(w.val)
}

func (w *webPortFlag) Set(s string) error {
	if s == "" {
		w.val = w.defaultPort
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if v <= 0 || v > 65535 {
		return fmt.Errorf("port must be 1-65535, got %d", v)
	}
	w.val = v
	return nil
}

func (w *webPortFlag) port() int { return w.val }

// newCameraFromConfig selects a camera implementation based on configuration.
func newCameraFromConfig(g gpio.Driver, cfg *config.Config) (camera.Camera, error) {
	switch cfg.Camera.Type {
	case "nikon_d90_gpio":
		return camera.NewNikonD90GPIO(
			g,
			cfg.Camera.FocusPin,
			cfg.Camera.ShutterPin,
			cfg.FocusDelay(),
			cfg.ShutterDelay(),
		), nil
	default:
		return nil, fmt.Errorf("unsupported camera type: %s", cfg.Camera.Type)
	}
}
