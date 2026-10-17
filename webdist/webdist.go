// Package webdist embeds the compiled frontend (web/dist) into the backend
// binary so a single container can serve both the REST API and the Web UI.
package webdist

import "embed"

// FS holds the built frontend. The Docker build copies web/dist into
// webdist/dist before compiling; without that copy the backend simply runs
// in API-only mode.
//
//go:embed all:dist
var FS embed.FS
