package web

import (
	"embed"
)

// static holds the embedded HTML, CSS, and JS files.
// The final binary includes all files under static/.
//
//go:embed static/*
var staticFiles embed.FS
