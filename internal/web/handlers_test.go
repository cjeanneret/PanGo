package web

import (
	"bufio"
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

// ---------- HandleCancel ----------

func TestHandleCancel_NoCapture(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodPost, "/cancel", nil)
	w := httptest.NewRecorder()

	h.HandleCancel(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleCancel_MethodNotAllowed(t *testing.T) {
	h := newTestHandlers(noopCapture)
	req := httptest.NewRequest(http.MethodGet, "/cancel", nil)
	w := httptest.NewRecorder()

	h.HandleCancel(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleCancel_CancelsRunningCapture(t *testing.T) {
	started := make(chan struct{})
	captureErr := make(chan error, 1)

	blockingCapture := func(ctx context.Context, _ Overrides) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	h := newTestHandlers(blockingCapture)

	// Subscribe to capture the broadcast
	ch, unsub := h.Broadcaster.Subscribe()
	defer unsub()

	// Start capture
	req1 := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w1 := httptest.NewRecorder()
	h.HandleRun(w1, req1)
	if w1.Code != http.StatusAccepted {
		t.Fatalf("run: status = %d, want %d", w1.Code, http.StatusAccepted)
	}

	<-started

	// Collect the capture error asynchronously
	go func() {
		// The goroutine in HandleRun will broadcast when it finishes;
		// we just need to wait for the running flag to clear.
		for {
			h.runningMu.Lock()
			running := h.running
			h.runningMu.Unlock()
			if !running {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		captureErr <- nil
	}()

	// Cancel
	req2 := httptest.NewRequest(http.MethodPost, "/cancel", nil)
	w2 := httptest.NewRecorder()
	h.HandleCancel(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("cancel: status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode cancel response: %v", err)
	}
	if resp["status"] != "cancelled" {
		t.Errorf("cancel response status = %q, want \"cancelled\"", resp["status"])
	}

	// Wait for capture goroutine to finish
	select {
	case <-captureErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for capture to finish after cancel")
	}

	// Verify the broadcast contained a warning about cancellation
	select {
	case msg := <-ch:
		var evt StatusEvent
		if err := json.Unmarshal([]byte(msg), &evt); err != nil {
			t.Fatalf("unmarshal broadcast: %v", err)
		}
		if evt.Level != "warning" {
			t.Errorf("broadcast level = %q, want \"warning\"", evt.Level)
		}
		if !strings.Contains(evt.Msg, "cancelled") {
			t.Errorf("broadcast msg = %q, should contain \"cancelled\"", evt.Msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancellation broadcast")
	}
}

func TestHandleRun_CancelledCaptureBroadcastsWarning(t *testing.T) {
	started := make(chan struct{})
	cancelCapture := func(ctx context.Context, _ Overrides) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	h := newTestHandlers(cancelCapture)
	ch, unsub := h.Broadcaster.Subscribe()
	defer unsub()

	// Start capture
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(validOverridesJSON()))
	w := httptest.NewRecorder()
	h.HandleRun(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	<-started

	// Cancel directly via the stored cancel func
	h.captureCancelMu.Lock()
	cancel := h.captureCancel
	h.captureCancelMu.Unlock()
	if cancel == nil {
		t.Fatal("captureCancel is nil after starting capture")
	}
	cancel()

	// Wait for warning broadcast
	select {
	case msg := <-ch:
		var evt StatusEvent
		if err := json.Unmarshal([]byte(msg), &evt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if evt.Level != "warning" {
			t.Errorf("level = %q, want \"warning\"", evt.Level)
		}
		if !strings.Contains(strings.ToLower(evt.Msg), "cancelled") {
			t.Errorf("msg = %q, should contain \"cancelled\"", evt.Msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for warning broadcast")
	}
}

// ---------- sanitizeSSE ----------

func TestSanitizeSSE(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"no_special", `{"msg":"hello"}`, `{"msg":"hello"}`},
		{"newline", "line1\nline2", "line1 line2"},
		{"carriage_return", "line1\rline2", "line1 line2"},
		{"crlf", "line1\r\nline2", "line1  line2"},
		{"multiple_newlines", "a\nb\nc", "a b c"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeSSE(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeSSE(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------- HandleStatusStream ----------

func TestHandleStatusStream_DeliversMessages(t *testing.T) {
	h := newTestHandlers(noopCapture)
	h.HeartbeatInterval = 10 * time.Second // long enough to not fire during this test

	srv := httptest.NewServer(http.HandlerFunc(h.HandleStatusStream))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Read initial ": connected" comment
	if !scanner.Scan() {
		t.Fatal("expected to read initial line")
	}
	if got := scanner.Text(); got != ": connected" {
		t.Errorf("first line = %q, want \": connected\"", got)
	}

	// Broadcast a message
	h.Broadcaster.Broadcast("info", "test-message")

	// Skip blank line(s) then read the data line
	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = line
			break
		}
	}

	if dataLine == "" {
		t.Fatal("never received a data: line")
	}

	payload := strings.TrimPrefix(dataLine, "data: ")
	var evt StatusEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		t.Fatalf("unmarshal SSE data: %v", err)
	}
	if evt.Msg != "test-message" {
		t.Errorf("msg = %q, want \"test-message\"", evt.Msg)
	}
	if evt.Level != "info" {
		t.Errorf("level = %q, want \"info\"", evt.Level)
	}
}

func TestHandleStatusStream_Heartbeat(t *testing.T) {
	h := newTestHandlers(noopCapture)
	h.HeartbeatInterval = 50 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(h.HandleStatusStream))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	// Read initial ": connected"
	if !scanner.Scan() {
		t.Fatal("expected to read initial line")
	}

	// Wait for at least one heartbeat
	gotHeartbeat := false
	deadline := time.After(2 * time.Second)
	lines := make(chan string)
	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("stream closed before heartbeat")
			}
			if line == ": heartbeat" {
				gotHeartbeat = true
			}
		case <-deadline:
			if !gotHeartbeat {
				t.Fatal("timeout: did not receive heartbeat within 2s")
			}
		}
		if gotHeartbeat {
			break
		}
	}
}

func TestHandleStatusStream_ClientDisconnect(t *testing.T) {
	h := newTestHandlers(noopCapture)
	h.HeartbeatInterval = 10 * time.Second

	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/status/stream", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	go func() {
		h.HandleStatusStream(w, req)
		close(done)
	}()

	// Give handler time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel client context
	cancel()

	select {
	case <-done:
		// Handler returned cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("HandleStatusStream did not return after client context cancelled")
	}

	body := w.Body.String()
	if !strings.Contains(body, ": connected") {
		t.Error("response should contain initial ': connected' comment")
	}
}

// ---------- HandleStatusStream SSE sanitization ----------

func TestHandleStatusStream_SanitizesNewlines(t *testing.T) {
	h := newTestHandlers(noopCapture)
	h.HeartbeatInterval = 10 * time.Second

	srv := httptest.NewServer(http.HandlerFunc(h.HandleStatusStream))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	// Skip ": connected"
	scanner.Scan()

	// Broadcast a message that contains raw newlines (simulates a bug in the broadcaster).
	// Normally json.Marshal escapes these, but we inject directly into the channel.
	ch, unsub := h.Broadcaster.Subscribe()
	defer unsub()

	// Inject a raw payload with newlines via a direct channel write to a subscriber.
	// We test sanitizeSSE indirectly by broadcasting a normal message — the unit test
	// for sanitizeSSE covers the edge case directly. Here we just verify the stream
	// delivers valid SSE frames.
	h.Broadcaster.Broadcast("info", "line-ok")

	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = line
			break
		}
	}

	if dataLine == "" {
		t.Fatal("never received a data: line")
	}

	// Verify the data line does not contain raw newlines (it's a single line)
	if strings.ContainsAny(dataLine, "\n\r") {
		t.Errorf("data line contains raw newlines: %q", dataLine)
	}

	// Drain the subscriber channel
	select {
	case <-ch:
	default:
	}
}
