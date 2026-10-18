package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var assets embed.FS

// Preview serves the explicitly synthetic V1 reference. E1 authentication and
// operational E2 screens are separate milestones, not simulated persistence.
func Preview() http.Handler {
	files, _ := fs.Sub(assets, "static")
	return http.FileServer(http.FS(files))
}
