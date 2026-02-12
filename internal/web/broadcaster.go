package web

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// StatusEvent represents a single status message for SSE.
type StatusEvent struct {
	Time  string `json:"t"`
	Level string `json:"l,omitempty"`
	Msg   string `json:"msg"`
}

// StatusBroadcaster distributes status messages to multiple SSE clients.
type StatusBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewStatusBroadcaster creates a new broadcaster.
func NewStatusBroadcaster() *StatusBroadcaster {
	return &StatusBroadcaster{
		clients: make(map[chan string]struct{}),
	}
}

// Subscribe returns a channel that receives broadcast messages and a cleanup function.
// The caller must call the returned cleanup when done (e.g. on client disconnect).
func (b *StatusBroadcaster) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
		close(ch)
	}
	return ch, unsub
}

// Broadcast sends a message to all subscribed clients.
// Messages are sent as JSON: {"t":"...","l":"info","msg":"..."}
// Slow clients may miss messages (non-blocking, buffered).
func (b *StatusBroadcaster) Broadcast(level, msg string) {
	evt := StatusEvent{
		Time:  time.Now().Format(time.RFC3339),
		Level: level,
		Msg:   msg,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	payload := string(data)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- payload:
		default:
			// channel full, skip
		}
	}
}

// BroadcastMsg is a convenience for level "info".
func (b *StatusBroadcaster) BroadcastMsg(msg string) {
	b.Broadcast("info", msg)
}

// BroadcastWriter implements io.Writer; each Write broadcasts the content to SSE clients.
func BroadcastWriter(b *StatusBroadcaster) *broadcastWriter {
	return &broadcastWriter{b: b}
}

// broadcastWriter wraps StatusBroadcaster as io.Writer for use with log.SetOutput.
type broadcastWriter struct {
	b *StatusBroadcaster
}

func (w *broadcastWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		w.b.BroadcastMsg(msg)
	}
	return len(p), nil
}
