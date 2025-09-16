package web

import (
	"embed"
	"io/fs"
)

// Assets embeds web templates and static files into the binary for single-exe distribution.
// Note: go:embed does not support recursive ** globs; embed concrete subdirectories instead.
// Directories are relative to this file (internal/web).
//
//go:embed ../../web/templates/*.html ../../web/static/css/* ../../web/static/js/*
var Assets embed.FS

// Sub returns a sub-filesystem rooted at the given path or panics (used at init wiring time).
func Sub(path string) fs.FS {
	sub, err := fs.Sub(Assets, path)
	if err != nil {
		panic(err)
	}
	return sub
}
