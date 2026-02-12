package web

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"
)

// Overrides holds capture parameters that can override config defaults.
type Overrides struct {
	HorizontalAngleDeg float64 `json:"horizontal_angle_deg"`
	VerticalAngleDeg   float64 `json:"vertical_angle_deg"`
	FocalLengthMm     float64 `json:"focal_length_mm"`
}

// RunCaptureFunc runs a capture with the given overrides.
// It is called from the POST /run handler in a goroutine.
type RunCaptureFunc func(ctx context.Context, overrides Overrides) error

// FormConfig holds default values for the capture form (from config).
type FormConfig struct {
	HorizontalAngleDeg float64 `json:"horizontal_angle_deg"`
	VerticalAngleDeg   float64 `json:"vertical_angle_deg"`
	FocalLengthMm      float64 `json:"focal_length_mm"`
}

// Handlers holds dependencies for HTTP handlers.
type Handlers struct {
	Broadcaster   *StatusBroadcaster
	RunCapture    RunCaptureFunc
	FormDefaults FormConfig
	runningMu     sync.Mutex
	running       bool
	staticFS      fs.FS
}

// NewHandlers creates handlers with the given dependencies.
// If runCapture is nil, POST /run will return 503 Service Unavailable.
func NewHandlers(broadcaster *StatusBroadcaster, runCapture RunCaptureFunc, formDefaults FormConfig, staticFS fs.FS) *Handlers {
	return &Handlers{
		Broadcaster:   broadcaster,
		RunCapture:    runCapture,
		FormDefaults:  formDefaults,
		staticFS:      staticFS,
	}
}

// HandleConfig returns the form default values (from config) as JSON.
func (h *Handlers) HandleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.FormDefaults)
}

// ServeIndex serves the main HTML page (root path only).
func (h *Handlers) ServeIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(h.staticFS, "index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// HandleRun handles POST /run to start a capture.
func (h *Handlers) HandleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var overrides Overrides
	if err := json.NewDecoder(r.Body).Decode(&overrides); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate
	if overrides.HorizontalAngleDeg <= 0 || overrides.HorizontalAngleDeg > 360 {
		http.Error(w, "horizontal_angle_deg must be between 1 and 360", http.StatusBadRequest)
		return
	}
	if overrides.VerticalAngleDeg <= 0 || overrides.VerticalAngleDeg > 180 {
		http.Error(w, "vertical_angle_deg must be between 1 and 180", http.StatusBadRequest)
		return
	}
	if overrides.FocalLengthMm <= 0 || overrides.FocalLengthMm > 500 {
		http.Error(w, "focal_length_mm must be between 1 and 500", http.StatusBadRequest)
		return
	}

	if h.RunCapture == nil {
		http.Error(w, "capture not configured", http.StatusServiceUnavailable)
		return
	}

	h.runningMu.Lock()
	if h.running {
		h.runningMu.Unlock()
		http.Error(w, "capture already in progress", http.StatusConflict)
		return
	}
	h.running = true
	h.runningMu.Unlock()

	// Run in goroutine; clear running when done
	go func() {
		defer func() {
			h.runningMu.Lock()
			h.running = false
			h.runningMu.Unlock()
		}()

		ctx := context.Background()
		if err := h.RunCapture(ctx, overrides); err != nil {
			h.Broadcaster.Broadcast("error", "Capture failed: "+err.Error())
			log.Printf("capture failed: %v", err)
		} else {
			h.Broadcaster.Broadcast("info", "Sequence complete")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// HandleStatusStream handles GET /status/stream for SSE.
func (h *Handlers) HandleStatusStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx

	ch, unsub := h.Broadcaster.Subscribe()
	defer unsub()

	// Send initial comment to establish connection
	w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	// Heartbeat while idle
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write([]byte("data: " + msg + "\n\n"))
			flusher.Flush()

		case <-ticker.C:
			w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
