package assets

import "embed"

//go:embed all:templates
var Templates embed.FS

//go:embed static
var Static embed.FS
