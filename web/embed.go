package web

import "embed"

//go:embed *.html
var StaticFiles embed.FS
