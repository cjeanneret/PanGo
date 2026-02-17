package web

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// ---------- ValidateOverrides ----------

func TestValidateOverrides_Valid(t *testing.T) {
	cases := []struct {
		name string
		o    Overrides
	}{
		{"mid_range", Overrides{180, 90, 35}},
		{"min_boundary", Overrides{1, 1, 1}},
		{"max_boundary", Overrides{360, 180, 500}},
		{"fractional", Overrides{0.5, 0.5, 0.5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOverrides(tc.o); err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
		})
	}
}

func TestValidateOverrides_ZeroRejected(t *testing.T) {
	cases := []struct {
		name string
		o    Overrides
	}{
		{"horizontal_zero", Overrides{0, 90, 35}},
		{"vertical_zero", Overrides{180, 0, 35}},
		{"focal_zero", Overrides{180, 90, 0}},
		{"all_zero", Overrides{0, 0, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOverrides(tc.o); err == nil {
				t.Error("expected error for zero value, got nil")
			}
		})
	}
}

func TestValidateOverrides_NaN(t *testing.T) {
	nan := math.NaN()
	cases := []struct {
		name string
		o    Overrides
	}{
		{"horizontal_NaN", Overrides{nan, 90, 35}},
		{"vertical_NaN", Overrides{180, nan, 35}},
		{"focal_NaN", Overrides{180, 90, nan}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOverrides(tc.o); err == nil {
				t.Error("expected error for NaN, got nil")
			}
		})
	}
}

func TestValidateOverrides_Infinity(t *testing.T) {
	posInf := math.Inf(1)
	negInf := math.Inf(-1)
	cases := []struct {
		name string
		o    Overrides
	}{
		{"horizontal_+Inf", Overrides{posInf, 90, 35}},
		{"horizontal_-Inf", Overrides{negInf, 90, 35}},
		{"vertical_+Inf", Overrides{180, posInf, 35}},
		{"focal_-Inf", Overrides{180, 90, negInf}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOverrides(tc.o); err == nil {
				t.Error("expected error for Infinity, got nil")
			}
		})
	}
}

func TestValidateOverrides_Negative(t *testing.T) {
	cases := []struct {
		name string
		o    Overrides
	}{
		{"horizontal_negative", Overrides{-1, 90, 35}},
		{"vertical_negative", Overrides{180, -5, 35}},
		{"focal_negative", Overrides{180, 90, -10}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOverrides(tc.o); err == nil {
				t.Error("expected error for negative value, got nil")
			}
		})
	}
}

func TestValidateOverrides_OutOfRange(t *testing.T) {
	cases := []struct {
		name string
		o    Overrides
	}{
		{"horizontal_361", Overrides{361, 90, 35}},
		{"vertical_181", Overrides{180, 181, 35}},
		{"focal_501", Overrides{180, 90, 501}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateOverrides(tc.o); err == nil {
				t.Error("expected error for out-of-range value, got nil")
			}
		})
	}
}

// ---------- Handler helpers ----------

func newTestHandlers(runCapture RunCaptureFunc) *Handlers {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>test</html>")},
	}
	return NewHandlers(
		NewStatusBroadcaster(),
		runCapture,
		FormConfig{
			HorizontalAngleDeg: 180,
			VerticalAngleDeg:   30,
			FocalLengthMm:      35,
		},
		staticFS,
	)
}

func noopCapture(_ context.Context, _ Overrides) error {
	return nil
}

func validOverridesJSON() []byte {
	data, _ := json.Marshal(Overrides{180, 30, 35})
	return data
}

// ---------- HandleRun ----------

func TestHandleRun_ValidPost(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleRun(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "started" {
		t.Errorf("response status = %q, want \"started\"", resp["status"])
	}

	// Wait for goroutine to finish
	time.Sleep(100 * time.Millisecond)
}

func TestHandleRun_GetMethodNotAllowed(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodGet, "/run", nil)
	w := httptest.NewRecorder()

	h.HandleRun(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleRun_InvalidJSON(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	h.HandleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_InvalidOverrides(t *testing.T) {
	h := newTestHandlers(noopCapture)
	data, _ := json.Marshal(Overrides{0, 90, 35})
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(data))
	w := httptest.NewRecorder()

	h.HandleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_OversizedBody(t *testing.T) {
	h := newTestHandlers(noopCapture)
	big := strings.Repeat("x", 2<<20) // 2 MB
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader(big))
	w := httptest.NewRecorder()

	h.HandleRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (oversized body)", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_NilRunCapture(t *testing.T) {
	h := newTestHandlers(nil)
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w := httptest.NewRecorder()

	h.HandleRun(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleRun_ConcurrentCapture(t *testing.T) {
	// Simulate a long-running capture
	started := make(chan struct{})
	blocking := make(chan struct{})
	slowCapture := func(_ context.Context, _ Overrides) error {
		close(started)
		<-blocking
		return nil
	}

	h := newTestHandlers(slowCapture)

	// First request starts capture
	req1 := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w1 := httptest.NewRecorder()
	h.HandleRun(w1, req1)
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first request: status = %d, want %d", w1.Code, http.StatusAccepted)
	}

	// Wait for goroutine to start
	<-started

	// Second request should be rejected as already running
	req2 := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w2 := httptest.NewRecorder()
	h.HandleRun(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("concurrent request: status = %d, want %d", w2.Code, http.StatusConflict)
	}

	close(blocking) // unblock first capture
	time.Sleep(100 * time.Millisecond)
}

func TestHandleRun_RateLimiting(t *testing.T) {
	h := newTestHandlers(noopCapture)

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w1 := httptest.NewRecorder()
	h.HandleRun(w1, req1)
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first request: status = %d, want %d", w1.Code, http.StatusAccepted)
	}

	// Wait a bit for goroutine to start and running flag to be cleared
	time.Sleep(200 * time.Millisecond)

	// Second request within 5 seconds should be rate-limited
	req2 := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w2 := httptest.NewRecorder()
	h.HandleRun(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("rate-limited request: status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

// ---------- HandleConfig ----------

func TestHandleConfig(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()

	h.HandleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var fc FormConfig
	if err := json.NewDecoder(w.Body).Decode(&fc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if fc.HorizontalAngleDeg != 180 {
		t.Errorf("HorizontalAngleDeg = %v, want 180", fc.HorizontalAngleDeg)
	}
	if fc.VerticalAngleDeg != 30 {
		t.Errorf("VerticalAngleDeg = %v, want 30", fc.VerticalAngleDeg)
	}
	if fc.FocalLengthMm != 35 {
		t.Errorf("FocalLengthMm = %v, want 35", fc.FocalLengthMm)
	}
}

// ---------- ServeIndex ----------

func TestServeIndex(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	if !strings.Contains(w.Body.String(), "<html>") {
		t.Error("body should contain HTML content")
	}
}
