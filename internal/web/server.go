package web

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"time"
)

// Server wraps the HTTP server and handlers.
type Server struct {
	addr     string
	handlers *Handlers
}

// NewServer creates a server configured for the given address and dependencies.
func NewServer(addr string, broadcaster *StatusBroadcaster, runCapture RunCaptureFunc, formDefaults FormConfig) *Server {
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("web: failed to sub static fs: %v", err)
	}

	handlers := NewHandlers(broadcaster, runCapture, formDefaults, subFS)

	return &Server{
		addr:     addr,
		handlers: handlers,
	}
}

// Mux returns an http.Handler with all routes registered.
func (s *Server) Mux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /run", s.handlers.HandleRun)
	mux.HandleFunc("GET /config", s.handlers.HandleConfig)
	mux.HandleFunc("GET /status/stream", s.handlers.HandleStatusStream)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.handlers.staticFS))))
	mux.HandleFunc("GET /{$}", s.handlers.ServeIndex) // exact match for root only

	return mux
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Printf("web server listening on %s", s.addr)
	return http.ListenAndServe(s.addr, s.Mux())
}

// Run starts the server and blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{Addr: s.addr, Handler: s.Mux()}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("web server listening on %s", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
