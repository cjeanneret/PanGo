package capture

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cjeanneret/PanGo/internal/hw/gpio"
	"github.com/cjeanneret/PanGo/internal/hw/stepper"
	"github.com/cjeanneret/PanGo/internal/logic/geometry"
	"github.com/cjeanneret/PanGo/internal/logic/motion"
)

// mockCamera records Shoot calls.
type mockCamera struct {
	mu    sync.Mutex
	shots int
}

func (m *mockCamera) Shoot() error {
	m.mu.Lock()
	m.shots++
	m.mu.Unlock()
	return nil
}

func (m *mockCamera) shotCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.shots
}

func newTestController() *motion.Controller {
	drv := &gpio.MockDriver{}
	pan := stepper.NewStepper(drv, stepper.Config{
		StepPin: 1, DirPin: 2, EnablePin: 3,
		StepsPerRev: 200, Microstepping: 16,
		StepDelay: 1 * time.Microsecond,
	})
	tilt := stepper.NewStepper(drv, stepper.Config{
		StepPin: 4, DirPin: 5, EnablePin: 6,
		StepsPerRev: 200, Microstepping: 16,
		StepDelay: 1 * time.Microsecond,
	})
	return motion.NewController(pan, tilt)
}

func TestInitializePosition(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:     2,
		TiltRows:       2,
		PanStepSize:    100,
		TiltStepSize:   50,
		StartPanAngle:  -90,
		StartTiltAngle: 15,
		StartPanSteps:  -800,
		StartTiltSteps: 133,
	}

	if err := seq.InitializePosition(plan); err != nil {
		t.Fatalf("InitializePosition: %v", err)
	}
}

func TestInitializePosition_ZeroSteps(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		StartPanSteps:  0,
		StartTiltSteps: 0,
	}

	if err := seq.InitializePosition(plan); err != nil {
		t.Fatalf("InitializePosition with zero steps: %v", err)
	}
}

func TestRunGridShot_1x1(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:   1,
		TiltRows:     1,
		PanStepSize:  100,
		TiltStepSize: 50,
	}

	ctx := context.Background()
	err := seq.RunGridShot(ctx, GridShotParams{
		GridPlan:      plan,
		Delay:         1 * time.Microsecond,
		MoveSpeed:     1 * time.Microsecond,
		ShotDelay:     1 * time.Microsecond,
		PostShotDelay: 1 * time.Microsecond,
	})
	if err != nil {
		t.Fatalf("RunGridShot: %v", err)
	}
	if cam.shotCount() != 1 {
		t.Errorf("shots = %d, want 1", cam.shotCount())
	}
}

func TestRunGridShot_2x2_ShotCount(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:   2,
		TiltRows:     2,
		PanStepSize:  100,
		TiltStepSize: 50,
	}

	ctx := context.Background()
	err := seq.RunGridShot(ctx, GridShotParams{
		GridPlan:      plan,
		Delay:         1 * time.Microsecond,
		MoveSpeed:     1 * time.Microsecond,
		ShotDelay:     1 * time.Microsecond,
		PostShotDelay: 1 * time.Microsecond,
	})
	if err != nil {
		t.Fatalf("RunGridShot: %v", err)
	}
	if cam.shotCount() != 4 {
		t.Errorf("shots = %d, want 4 (2x2)", cam.shotCount())
	}
}

func TestRunGridShot_3x4_ShotCount(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:   3,
		TiltRows:     4,
		PanStepSize:  100,
		TiltStepSize: 50,
	}

	ctx := context.Background()
	err := seq.RunGridShot(ctx, GridShotParams{
		GridPlan:      plan,
		Delay:         1 * time.Microsecond,
		MoveSpeed:     1 * time.Microsecond,
		ShotDelay:     1 * time.Microsecond,
		PostShotDelay: 1 * time.Microsecond,
	})
	if err != nil {
		t.Fatalf("RunGridShot: %v", err)
	}
	if cam.shotCount() != 12 {
		t.Errorf("shots = %d, want 12 (3x4)", cam.shotCount())
	}
}

func TestRunGridShot_ContextCancellation(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:   100,
		TiltRows:     100,
		PanStepSize:  10,
		TiltStepSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	err := seq.RunGridShot(ctx, GridShotParams{
		GridPlan:      plan,
		Delay:         1 * time.Microsecond,
		MoveSpeed:     1 * time.Microsecond,
		ShotDelay:     1 * time.Microsecond,
		PostShotDelay: 1 * time.Microsecond,
	})

	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
	// Should have taken far fewer than 100*100=10000 shots
	if cam.shotCount() >= 10000 {
		t.Errorf("expected fewer shots due to cancellation, got %d", cam.shotCount())
	}
}

func TestRunGridShot_ContextCancelMidSequence(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:   10,
		TiltRows:     10,
		PanStepSize:  10,
		TiltStepSize: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := seq.RunGridShot(ctx, GridShotParams{
		GridPlan:      plan,
		Delay:         10 * time.Millisecond,
		MoveSpeed:     1 * time.Microsecond,
		ShotDelay:     1 * time.Microsecond,
		PostShotDelay: 1 * time.Microsecond,
	})

	if err == nil {
		t.Error("expected context deadline error, got nil")
	}
	// Should have taken some but not all shots
	shots := cam.shotCount()
	if shots == 0 {
		t.Error("expected at least some shots before cancellation")
	}
	if shots >= 100 {
		t.Errorf("expected fewer than 100 shots due to cancellation, got %d", shots)
	}
}

func TestRunGridShot_LargeGrid(t *testing.T) {
	ctrl := newTestController()
	cam := &mockCamera{}
	seq := NewSequence(ctrl, cam)

	plan := &geometry.GridPlan{
		PanColumns:   5,
		TiltRows:     7,
		PanStepSize:  50,
		TiltStepSize: 30,
	}

	ctx := context.Background()
	err := seq.RunGridShot(ctx, GridShotParams{
		GridPlan:      plan,
		Delay:         1 * time.Microsecond,
		MoveSpeed:     1 * time.Microsecond,
		ShotDelay:     1 * time.Microsecond,
		PostShotDelay: 1 * time.Microsecond,
	})
	if err != nil {
		t.Fatalf("RunGridShot: %v", err)
	}
	if cam.shotCount() != 35 {
		t.Errorf("shots = %d, want 35 (5x7)", cam.shotCount())
	}
}
