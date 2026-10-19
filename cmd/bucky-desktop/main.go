package main

import (
	"context"
	"log"
	"runtime"

	"github.com/amigoer/bucky/frontend"
	"github.com/amigoer/bucky/internal/app"
	"github.com/amigoer/bucky/internal/macwindow"
	"github.com/amigoer/bucky/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Keep in step with --titlebar-height in frontend/src/style.css.
const (
	titleBarHeight   = 52.0
	trafficLightLeft = 18.0
)

func main() {
	backend, err := service.New(context.Background(), service.Options{})
	if err != nil {
		log.Fatal(err)
	}
	desktop := application.New(application.Options{
		Name: "Bucky", Description: "Cloudflare R2 object storage client",
		Services: []application.Service{application.NewService(app.New(backend))},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(frontend.Assets)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	window := desktop.Window.NewWithOptions(newWindowOptions())
	alignTrafficLights(window)
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		desktop.Event.Emit("files:dropped", event.Context().DroppedFiles())
	})
	if err := desktop.Run(); err != nil {
		_ = backend.Close()
		log.Fatal("Desktop application failed to start")
	}
}

func newWindowOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Title: "Bucky", Width: 1280, Height: 840, MinWidth: 960, MinHeight: 640,
		URL: "/", EnableFileDrop: true,
		// macOS keeps the native frame so its traffic lights stay available;
		// the renderer draws the rest of the title bar on every platform.
		Frameless:        runtime.GOOS != "darwin",
		BackgroundColour: application.NewRGB(255, 255, 255),
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHidden,
		},
	}
}

func alignTrafficLights(window *application.WebviewWindow) {
	if runtime.GOOS != "darwin" {
		return
	}
	window.OnWindowEvent(events.Mac.WindowDidBecomeKey, func(*application.WindowEvent) {
		macwindow.SetTrafficLightPosition(
			window.NativeWindow(), trafficLightLeft, titleBarHeight/2)
	})
}
