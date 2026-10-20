package desktop

import "github.com/egoist/mygo/ui"

// entryMenu builds the menu of one row. The menu of a row inside a
// multiple choice acts on the whole choice.
func (a *App) entryMenu(c *ui.Context, m *ui.Menu, row entry) {
	choice := []entry{row}
	if a.browse.selection.Has(row.key) && a.browse.selection.Len() > 1 {
		choice = a.selectedEntries()
	}
	single := len(choice) == 1

	open := a.t("menu.quickLook")
	if row.folder && single {
		open = a.t("menu.open")
	}
	if m.Item(open).Chosen() {
		if single {
			a.openEntry(row)
		}
	}
	if m.Item(a.t("menu.download")).Chosen() {
		a.download(choice)
	}
	if m.Item(a.t("menu.share")).Chosen() {
		a.openShare(choice)
	}
	m.Separator()
	if m.Item(a.t("menu.copyKey")).Chosen() {
		keys := make([]string, 0, len(choice))
		for _, item := range choice {
			keys = append(keys, item.key)
		}
		a.copyTextCurrent(joinLines(keys))
	}
	if a.readOnly() {
		return
	}
	if m.Item(a.t("menu.copyTo")).Chosen() {
		a.openPlan("copy", choice)
	}
	if m.Item(a.t("menu.moveTo")).Chosen() {
		a.openPlan("move", choice)
	}
	m.Separator()
	if m.Item(a.t("menu.delete")).Chosen() {
		a.openPlan("delete", choice)
	}
}

// selectionBar floats the actions of the current choice over the content.
func (a *App) selectionBar(c *ui.Context) {
	choice := a.selectedEntries()
	if len(choice) == 0 {
		return
	}
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Left(0).Right(0).Bottom(16).Justify(ui.Center).Children(func() {
			bar := ui.Row(c).Padding(4, 6, 4, 14).Radius(12).Gap(2).
				AlignItems(ui.Center).Background(a.pal.hud).
				Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.28))
			bar.Children(func() {
				ui.Text(c, a.tn("selection.summary", len(choice), "count", len(choice))).
					FontSize(12).TextColor(ui.Hex("#ffffff")).Padding(0, 6, 0, 4)
				barSeparator(c)
				a.barButton(c, iconDownload, a.t("menu.download"), false, func() {
					a.download(choice)
				})
				a.barButton(c, iconShare, a.t("menu.share"), false, func() {
					a.openShare(choice)
				})
				if !a.readOnly() {
					barSeparator(c)
					a.barButton(c, iconArrowUpRight, a.t("selection.copy"), false, func() {
						a.openPlan("copy", choice)
					})
					a.barButton(c, iconArrowRightLeft, a.t("selection.move"), false, func() {
						a.openPlan("move", choice)
					})
					a.barButton(c, iconTrash, a.t("common.delete"), true, func() {
						a.openPlan("delete", choice)
					})
				}
				barSeparator(c)
				a.barButton(c, iconX, a.t("selection.clear"), false, func() {
					a.clearSelection()
				})
			})
		})
	})
}

// barButton is one action of the floating selection bar.
func (a *App) barButton(c *ui.Context, glyph *ui.SVG, label string, destructive bool, action func()) {
	button := ui.Button(c, "").Label(label).Tooltip(label).Padding(4, 8).Radius(6)
	button.Children(func() {
		ui.Icon(c, glyph).FontSize(14).TextColor(ui.Hex("#ffffff"))
		ui.Text(c, label).FontSize(12).TextColor(ui.Hex("#ffffff")).SingleLine()
	})
	if destructive && button.Hovered() {
		button.Background(a.pal.destructive)
	}
	if button.Clicked() {
		action()
	}
}

func barSeparator(c *ui.Context) {
	ui.Box(c).Width(1).Height(16).Margin(0, 4).Background(ui.RGBA(255, 255, 255, 0.16))
}

// joinLines joins keys with newlines for the clipboard.
func joinLines(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += "\n"
		}
		out += value
	}
	return out
}
