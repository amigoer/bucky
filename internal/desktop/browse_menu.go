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
	var size int64
	for _, row := range choice {
		size += row.size
	}
	readOnly := a.readOnly()
	white := ui.Hex("#f5f5f7")
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Left(0).Right(0).Bottom(22).Justify(ui.Center).Children(func() {
			bar := ui.Row(c).Padding(5).Radius(14).Gap(2).AlignItems(ui.Center).
				Background(a.pal.hud).Border(0.5, ui.RGBA(255, 255, 255, 0.08)).
				Shadow(0, 14, 36, 0, ui.RGBA(0, 0, 0, 0.3))
			bar.Children(func() {
				ui.Row(c).Gap(4).Padding(0, 12, 0, 10).AlignItems(ui.Center).Children(func() {
					count := a.tn("browser.items", len(choice), "count", formatCount(int64(len(choice))))
					ui.Text(c, count).FontSize(13).FontWeight(600).TextColor(white).SingleLine()
					ui.Text(c, "· "+formatBytes(size)).FontSize(13).
						TextColor(ui.RGBA(255, 255, 255, 0.55)).SingleLine()
				})
				barSeparator(c)
				a.barButton(c, iconDownload, a.t("menu.download"), barFilled, false, func() {
					a.download(choice)
				})
				a.barButton(c, iconLink, a.t("menu.share"), barPlain, false, func() {
					a.openShare(choice)
				})
				a.barButton(c, iconCopy, a.t("selection.copy"), barPlain, readOnly, func() {
					a.openPlan("copy", choice)
				})
				a.barButton(c, iconArrowRightLeft, a.t("selection.move"), barPlain, readOnly, func() {
					a.openPlan("move", choice)
				})
				a.barButton(c, iconTrash, a.t("common.delete"), barDestructive, readOnly, func() {
					a.openPlan("delete", choice)
				})
				barSeparator(c)
				label := a.t("selection.clear")
				clear := ui.ButtonBase(c).Label(label).Tooltip(label).Size(32, 32).Radius(9)
				color := ui.RGBA(255, 255, 255, 0.6)
				if clear.Hovered() {
					clear.Background(ui.RGBA(255, 255, 255, 0.1))
					color = ui.Hex("#ffffff")
				}
				clear.Children(func() { ui.Icon(c, iconX).FontSize(15).TextColor(color) })
				if clear.Clicked() {
					a.clearSelection()
				}
			})
		})
	})
}

// The looks of a selection bar button: the first action is filled, delete
// is red.
const (
	barPlain = iota
	barFilled
	barDestructive
)

// barButton is one action of the floating selection bar. One a read-only
// connection does not allow stays in place, dimmed, and says why.
func (a *App) barButton(c *ui.Context, glyph *ui.SVG, label string, look int, disabled bool, action func()) {
	button := ui.ButtonBase(c).Label(label).Height(32).PaddingX(11).Radius(9).Disabled(disabled)
	color := ui.Hex("#f5f5f7")
	rest, hover := ui.Transparent, ui.RGBA(255, 255, 255, 0.1)
	switch look {
	case barFilled:
		rest, hover = ui.RGBA(255, 255, 255, 0.1), ui.RGBA(255, 255, 255, 0.16)
	case barDestructive:
		color, hover = ui.Oklch(0.76, 0.14, 27), ui.Oklch(0.6, 0.2, 27).Alpha(0.22)
	}
	button.Background(rest)
	if disabled {
		button.Opacity(0.4).Tooltip(a.t("browser.readOnlyTip"))
	} else if button.Hovered() {
		button.Background(hover)
	}
	button.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, glyph).FontSize(15).TextColor(color)
			ui.Text(c, label).FontSize(13).TextColor(color).SingleLine()
		})
	})
	if !disabled && button.Clicked() {
		action()
	}
}

func barSeparator(c *ui.Context) {
	ui.Box(c).Width(1).Height(18).Margin(0, 4).Background(ui.RGBA(255, 255, 255, 0.16))
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
