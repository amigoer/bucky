//go:build !darwin || !cgo

// Package macwindow adjusts native macOS window chrome that Wails does not
// expose. On other platforms and without cgo every function is a no-op.
package macwindow

import "unsafe"

// SetTrafficLightPosition does nothing where native window buttons do not
// overlap the renderer's title bar or cgo is unavailable.
func SetTrafficLightPosition(_ unsafe.Pointer, _, _ float64) {}
