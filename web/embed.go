// Package web menyatukan hasil build Vite ke dalam binary.
//
// Berkas ini berada di dalam folder proyek frontend karena go:embed tidak bisa
// menjangkau direktori di luar paketnya. Inilah yang membuat seluruh aplikasi —
// API dan SPA — menjadi satu berkas yang bisa disalin, tanpa nginx terpisah
// (README §13.4).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func FS() (fs.FS, error) { return fs.Sub(dist, "dist") }
