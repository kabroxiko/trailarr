package assets

import (
	"embed"
)

//go:embed dist/* dist/assets/*
var EmbeddedDist embed.FS
