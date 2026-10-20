// Command bucky-desktop is the Bucky desktop application. Its window is
// drawn by MyGo's native UI toolkit, so it starts no webview and needs no
// frontend build.
package main

import (
	"context"
	"log"

	"github.com/amigoer/bucky/internal/desktop"
	"github.com/amigoer/bucky/internal/service"
	"github.com/egoist/mygo"
)

func main() {
	ctx := context.Background()
	backend, err := service.New(ctx, service.Options{})
	if err != nil {
		log.Fatalf("Bucky cannot start: %v", err)
	}
	app, err := desktop.New(ctx, backend)
	if err != nil {
		_ = backend.Close()
		log.Fatalf("Bucky cannot start: %v", err)
	}
	mygo.App.SetName("Bucky")
	mygo.App.SetVersion(desktop.Version)
	mygo.App.SetMenu(appMenu())
	mygo.App.OnQuit(app.Close)
	if err := app.Run(); err != nil {
		app.Close()
		log.Fatalf("Bucky stopped: %v", err)
	}
}

// appMenu is the menu bar: the application menu, the editing commands text
// fields use, and the window commands.
func appMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{
			Label: "Bucky",
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleAbout},
				mygo.Separator(),
				{Role: mygo.RoleHide},
				{Role: mygo.RoleHideOthers},
				{Role: mygo.RoleUnhide},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			},
		},
		{
			Label: "Edit",
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleUndo},
				{Role: mygo.RoleRedo},
				mygo.Separator(),
				{Role: mygo.RoleCut},
				{Role: mygo.RoleCopy},
				{Role: mygo.RolePaste},
				{Role: mygo.RoleSelectAll},
			},
		},
		{
			Label: "Window",
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleMinimize},
				{Role: mygo.RoleZoom},
				mygo.Separator(),
				{Role: mygo.RoleClose},
			},
		},
	})
}
