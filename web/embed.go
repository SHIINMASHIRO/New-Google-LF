// Package web embeds the compiled frontend assets.
package web

import "embed"

//go:embed dist
var Assets embed.FS
