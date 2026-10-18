//go:build darwin && cgo

// Package macwindow adjusts native macOS window chrome that Wails does not
// expose. On other platforms and without cgo every function is a no-op.
package macwindow

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa
#include "trafficlight_darwin.h"
*/
import "C"

import "unsafe"

// SetTrafficLightPosition centres the window buttons in the renderer's title
// bar. The position is re-applied after resizes and fullscreen exits, when
// AppKit resets it.
func SetTrafficLightPosition(nsWindow unsafe.Pointer, left, centreY float64) {
	if nsWindow == nil {
		return
	}
	C.BUCKYSetTrafficLightPosition(nsWindow, C.double(left), C.double(centreY))
}
