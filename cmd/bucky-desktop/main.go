package main

import (
	"context"
	"log"

	"github.com/amigoer/bucky/frontend"
	"github.com/amigoer/bucky/internal/app"
	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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
	backend.Transfers.OnUpdate = func(task core.TransferTask) {
		desktop.Event.Emit("transfer:updated", task)
	}
	window := desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Bucky", Width: 1280, Height: 840, MinWidth: 960, MinHeight: 640,
		URL: "/", EnableFileDrop: true,
		BackgroundColour: application.NewRGB(246, 247, 244),
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		desktop.Event.Emit("files:dropped", event.Context().DroppedFiles())
	})
	if err := desktop.Run(); err != nil {
		_ = backend.Close()
		log.Fatal("Desktop application failed to start")
	}
}
