// Package static provides embedded static assets for the dashboard.
package static

import "embed"

//go:embed css/*.css js/*.js img/*
var FS embed.FS
