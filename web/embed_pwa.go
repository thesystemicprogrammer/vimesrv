package web

import "embed"

// PWAFiles embeds the Angular PWA application
// The PWA is built to web/pwa/dist/vimesrv-client/browser/ by Angular
//
//go:embed pwa/dist/vimesrv-client/browser
var PWAFiles embed.FS
