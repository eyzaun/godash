package main

import (
	"embed"
	"io/fs"
)

// Embed web UI assets so the program runs as a single executable without external files.
//
//go:embed web/templates web/static
var embeddedAssets embed.FS

// getTemplatesFS returns an fs.FS view rooted at web/templates
func getTemplatesFS() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "web/templates")
	if err != nil {
		panic(err)
	}
	return sub
}

// getStaticFS returns an fs.FS view rooted at web/static
func getStaticFS() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "web/static")
	if err != nil {
		panic(err)
	}
	return sub
}
