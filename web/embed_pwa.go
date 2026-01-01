package web

import "embed"

// PWAFiles embeds the Angular PWA application
// The PWA is built to web/pwa/dist/pwa/browser/ by Angular
//
//go:embed pwa/dist/pwa/browser
var PWAFiles embed.FS
