package camera

// Camera is the high-level interface used by the rest of the application.
// It represents an abstract "camera", regardless of how it's controlled
// (GPIO, USB, network protocol, etc.).
type Camera interface {
	// Shoot triggers a single photo capture (simple mode).
	Shoot() error
}
