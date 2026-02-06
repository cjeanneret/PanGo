package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfgPath := filepath.Join("configs", "default.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	// Initialize debug system
	debug.Init(cfg.Defaults.DebugLevel)
	debug.Section("Initialization")
	debug.Value("Config path", cfgPath)
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
	panMotor := stepper.NewStepper(gpioDriver, stepper.Config{
		StepPin:       cfg.PanStepper.StepPin,
		DirPin:        cfg.PanStepper.DirPin,
		StepsPerRev:   cfg.PanStepper.StepsPerRev,
		Microstepping: cfg.PanStepper.Microstepping,
	})
	debug.PrintStruct("Pan stepper config", cfg.PanStepper)
	tiltMotor := stepper.NewStepper(gpioDriver, stepper.Config{
		StepPin:       cfg.TiltStepper.StepPin,
		DirPin:        cfg.TiltStepper.DirPin,
		StepsPerRev:   cfg.TiltStepper.StepsPerRev,
		Microstepping: cfg.TiltStepper.Microstepping,
	})
	debug.PrintStruct("Tilt stepper config", cfg.TiltStepper)

	// Initialize camera via configurable type
	debug.Step(3, "Initializing camera")
	cam, err := newCameraFromConfig(gpioDriver, cfg)
	if err != nil {
		log.Fatalf("init camera failed: %v", err)
	}
	debug.Value("Camera type", cfg.Camera.Type)
	debug.Value("Focus pin", cfg.Camera.FocusPin)
	debug.Value("Shutter pin", cfg.Camera.ShutterPin)

	// Calculate photo grid
	debug.Step(4, "Calculating grid plan")
	fovCalc, err := geometry.NewFOVCalculator(cfg)
	if err != nil {
		log.Fatalf("create FOV calculator failed: %v", err)
	}
	stepsCalc := geometry.NewStepsCalculator(cfg)
	gridPlan := geometry.CalculateGridPlan(cfg, fovCalc, stepsCalc)

	// Level 1: important grid info
	totalPhotos := gridPlan.PanColumns * gridPlan.TiltRows
	debug.Summary("Grid Plan Summary")
	debug.Grid(gridPlan.PanColumns, gridPlan.TiltRows, totalPhotos)
	debug.Info("Step sizes: pan=%d steps, tilt=%d steps", gridPlan.PanStepSize, gridPlan.TiltStepSize)

	// Level 3: verbose details
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

	// High-level controllers
	debug.Step(5, "Creating motion and capture controllers")
	motionCtrl := motion.NewController(panMotor, tiltMotor)
	captureSeq := capture.NewSequence(motionCtrl, cam)

	// Execute grid shot sequence with automatic calculation
	debug.Section("Starting Grid Shot Sequence")
	err = captureSeq.RunGridShot(ctx, capture.GridShotParams{
		GridPlan: gridPlan,

		Delay:         500 * time.Millisecond,
		MoveSpeed:     cfg.MoveSpeed(),
		ShotDelay:     300 * time.Millisecond,
		PostShotDelay: cfg.PostShotDelay(),
	})
	if err != nil {
		log.Fatalf("scenario failed: %v", err)
	}

	debug.Section("Sequence Complete")
}

// newCameraFromConfig selects a camera implementation based on configuration.
// To add a new type, simply extend this switch without changing the rest
// of the application.
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
