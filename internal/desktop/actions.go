package desktop

import (
	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// download queues the chosen rows, asking where they go.
func (a *App) download(choice []entry) {
	if a.location == nil || len(choice) == 0 {
		return
	}
	location := *a.location
	objects := make([]entry, 0, len(choice))
	for _, row := range choice {
		if !row.folder {
			objects = append(objects, row)
		}
	}
	if len(choice) == 1 && len(objects) == 1 {
		a.downloadSingle(location, objects[0])
		return
	}
	a.downloadMany(location, choice)
}

// downloadSingle asks for a file name and queues one download.
func (a *App) downloadSingle(location Location, row entry) {
	go func() {
		target, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.win, Title: a.t("common.download"), DefaultPath: basename(row.key),
		})
		if err != nil || target == "" {
			return
		}
		_, err = a.svc.StartTransfer(a.ctx, core.TransferRequest{
			Direction: "download",
			Object:    core.ObjectRef{ProfileID: location.ProfileID, Bucket: location.Bucket, Key: row.key},
			LocalPath: target,
		})
		a.update(func() {
			if err != nil {
				a.toastCurrent(a.errorText(err))
				return
			}
			a.toastCurrent(a.tn("toast.queued", 1, "count", 1))
			a.loadTasks()
		})
	}()
}

// downloadMany asks for a directory and queues everything below it.
func (a *App) downloadMany(location Location, choice []entry) {
	go func() {
		dir, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.win, Title: a.t("common.download"), Directory: true,
		})
		if err != nil || len(dir) == 0 {
			return
		}
		refs, err := a.expandEntries(a.ctx, location, choice)
		if err != nil {
			a.update(func() { a.toastCurrent(a.errorText(err)) })
			return
		}
		if len(refs) == 0 {
			a.update(func() { a.toastCurrent(a.t("toast.nothingToDownload")) })
			return
		}
		results, err := a.downloadTo(a.ctx, dir[0], refs, location.Prefix)
		a.update(func() {
			if err != nil {
				a.toastCurrent(a.errorText(err))
				return
			}
			failed := 0
			queued := 0
			var first *core.Error
			for _, result := range results {
				if result.err != nil {
					failed++
					if first == nil {
						first = result.err
					}
					continue
				}
				if result.task != nil {
					queued++
				}
			}
			switch {
			case failed > 0:
				message := ""
				if first != nil {
					message = a.errorText(first)
				}
				a.toastCurrent(a.tf("toast.downloadPartial", "failed", failed,
					"total", len(results), "message", message))
			case queued > 0:
				a.toastCurrent(a.tn("toast.queued", queued, "count", queued))
			}
			a.loadTasks()
		})
	}()
}

// confirmOverlay asks about something that cannot be undone.
func (a *App) confirmOverlay(c *ui.Context) {
	state := &a.confirm
	if !state.open {
		return
	}
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Background(a.pal.scrim).Modal()
		backdrop.Children(func() {
			panel := ui.Column(c).Width(400).Gap(14).Padding(22).Radius(16).
				Background(a.pal.background).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.22))
			panel.Children(func() {
				ui.Text(c, state.title).FontSize(14).Bold()
				ui.Text(c, state.message).FontSize(12.5).TextColor(a.pal.muted)
				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, a.t("common.cancel")).Clicked() {
						a.confirm = confirmState{}
					}
					button := ui.PrimaryButton(c, state.confirm)
					if state.destructive {
						button.Background(a.pal.destructive)
					}
					if button.Clicked() {
						action := state.action
						a.confirm = confirmState{}
						if action != nil {
							action()
						}
					}
				})
			})
			if backdrop.OverlayShortcut(0, ui.KeyEscape) {
				a.confirm = confirmState{}
			}
		})
	})
}

// shortcutKeys wires the keyboard shortcuts of the whole window.
func (a *App) shortcuts(c *ui.Context) {
	if a.overlayOpen() {
		return
	}
	mod := ui.Cmd
	if c.Shortcut(mod, ui.KeyF) || c.Shortcut(mod, ui.KeyK) {
		a.page = pageBrowse
		a.openPalette()
		a.search.focus.Focus()
	}
	if c.Shortcut(mod, ui.KeyComma) {
		a.page = pageSettings
		a.loadSettings()
	}
	if c.Shortcut(mod|ui.Shift, ui.KeyT) {
		a.page = pageTransfers
		a.loadTasks()
	}
	if c.Shortcut(mod|ui.Alt, ui.KeyU) {
		a.openQuick()
	}
	if a.page == pageBrowse {
		if c.Shortcut(mod, ui.KeyBracketLeft) {
			a.goBack()
		}
		if c.Shortcut(mod, ui.KeyBracketRight) {
			a.goForward()
		}
	}
}

// overlayOpen reports whether a panel is covering the window.
func (a *App) overlayOpen() bool {
	return a.preview.open || a.share.open || a.plan.open || a.connect.open || a.quick.open ||
		a.confirm.open
}
