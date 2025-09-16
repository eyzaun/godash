package main

import (
	"embed"
)

// Embed web UI assets so the program runs as a single executable without external files.
//
//go:embed web/templates web/static
var embeddedAssets embed.FS
