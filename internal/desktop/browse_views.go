package desktop

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// The large folder of the grid: a filled shape under an outline of the same
// path, so the two take their own colors.
var (
	folderPath    = `M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z`
	folderFill    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="` + folderPath + `" fill="currentColor"/></svg>`))
	folderOutline = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="` + folderPath + `" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linejoin="round" stroke-linecap="round"/></svg>`))
)

// List view column widths, as the design sets them.
const (
	listSizeWidth = 92
	listTimeWidth = 150
	listKindWidth = 80
)

// listView shows the folder as rows under a header that sorts them.
func (a *App) listView(c *ui.Context) {
	state := &a.browse
	ui.Column(c).Fill().Children(func() {
		a.listHeader(c)
		state.list.Key = func(row int) any { return state.shown[row].key }
		state.list.Label = func(row int) string { return state.shown[row].name }
		state.list.Selection = &state.selection
		state.list.Selected = &state.selectedIndex
		state.list.MenuKeepsSelection = false
		list := ui.List(c, &state.list, len(state.shown), func(i int) {
			a.listRow(c, state.shown[i])
		}).Grow(1).Padding(6, 12, 80, 12).Gap(1).Label(a.t("browser.colName"))
		if list.Submitted() {
			if state.selectedIndex >= 0 && state.selectedIndex < len(state.shown) {
				a.openEntry(state.shown[state.selectedIndex])
			}
		}
	})
}

// listHeader names the columns; the name, size and time sort the rows.
func (a *App) listHeader(c *ui.Context) {
	ui.Row(c).Height(30).PaddingX(24).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		a.headerCell(c, a.t("browser.colName"), "name", 0, false)
		a.headerCell(c, a.t("browser.colSize"), "size", listSizeWidth, true)
		a.headerCell(c, a.t("browser.colModified"), "modified", listTimeWidth, false)
		ui.Text(c, a.t("browser.colKind")).FontSize(11.5).FontWeight(500).
			TextColor(a.pal.muted).Width(listKindWidth).SingleLine()
	})
}

// headerCell is one column title; clicking it sorts by the column, or turns
// the order round when it already does.
func (a *App) headerCell(c *ui.Context, title, key string, width float32, end bool) {
	active := a.browse.sortKey == key
	cell := ui.Row(c).Gap(4).AlignItems(ui.Center).Label(title)
	if width > 0 {
		cell.Width(width)
	} else {
		cell.Grow(1).MinWidth(0)
	}
	if end {
		cell.Justify(ui.End).PaddingEnd(18)
	}
	color := a.pal.muted
	if active {
		color = a.pal.text
	}
	cell.Children(func() {
		ui.Text(c, title).FontSize(11.5).FontWeight(500).TextColor(color).SingleLine()
		arrow := ui.Icon(c, iconChevronUp).FontSize(10).TextColor(color)
		if !active {
			arrow.Opacity(0)
		} else if a.browse.sortDir < 0 {
			arrow.Rotate(180)
		}
	})
	if cell.Clicked() {
		direction := 1
		if active {
			direction = -a.browse.sortDir
		}
		a.setSort(key, direction)
	}
}

// listRow builds one row of the list view.
func (a *App) listRow(c *ui.Context, row entry) {
	selected := a.browse.selection.Has(row.key)
	text, muted := a.pal.text, a.pal.muted
	kinds := a.pal.kindPaletteOf(row.kind())
	tileBg, tileFg := kinds.background, kinds.foreground
	if selected {
		text, muted = a.pal.onPrimary, a.pal.onPrimaryMuted
		tileBg, tileFg = ui.RGBA(255, 255, 255, 0.22), a.pal.onPrimary
	}
	item := ui.Row(c).Height(36).PaddingX(12).AlignItems(ui.Center).Radius(8)
	if selected {
		item.Background(a.pal.primary)
	}
	item.Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(11).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(28, 28).Radius(7).Center().Background(tileBg).Children(func() {
				ui.Icon(c, iconOf(row.kind())).FontSize(15).TextColor(tileFg)
			})
			name := ui.Text(c, row.name).FontSize(13).SingleLine().TextColor(text)
			if selected {
				name.FontWeight(500)
			}
		})
		size, modified := "—", "—"
		if !row.folder {
			size, modified = formatBytes(row.size), formatDate(row.modified)
		}
		ui.Text(c, size).FontSize(12.5).TextAlign(ui.End).SingleLine().TextColor(muted).
			Width(listSizeWidth).PaddingEnd(18)
		ui.Text(c, modified).FontSize(12.5).SingleLine().TextColor(muted).Width(listTimeWidth)
		ui.Text(c, a.t("kinds."+row.kind())).FontSize(12.5).SingleLine().TextColor(muted).
			Width(listKindWidth)
	})
	item.ContextMenu(func(m *ui.Menu) { a.entryMenu(c, m, row) })
	if item.DoubleClicked() {
		a.openEntry(row)
	}
}

// gridView shows the folder as a grid of tiles.
func (a *App) gridView(c *ui.Context) {
	state := &a.browse
	state.grid.Key = func(item int) any { return state.shown[item].key }
	state.grid.Label = func(item int) string { return state.shown[item].name }
	state.grid.Selection = &state.selection
	state.grid.Selected = &state.selectedIndex

	grid := ui.GridView(c, &state.grid, len(state.shown), 112, 120, func(i int) {
		a.gridTile(c, state.shown[i])
	}).Fill().Padding(14, 18, 80, 18)
	if grid.Submitted() {
		if state.selectedIndex >= 0 && state.selectedIndex < len(state.shown) {
			a.openEntry(state.shown[state.selectedIndex])
		}
	}
}

// gridTile draws one tile of the grid view.
func (a *App) gridTile(c *ui.Context, row entry) {
	selected := a.browse.selection.Has(row.key)
	tile := ui.Column(c).Fill().Gap(6).Padding(6, 0, 0, 0).Radius(11).AlignItems(ui.Center)
	if !selected && tile.Hovered() {
		tile.Background(a.pal.hover)
	}
	tile.Children(func() {
		a.gridThumb(c, row, selected)
		label := ui.Text(c, row.name).FontSize(12).LineHeight(1.42).TextAlign(ui.Center).
			SingleLine().MaxWidth(104).Padding(1, 6).Radius(5)
		if selected {
			label.Background(a.pal.primary).TextColor(a.pal.onPrimary).FontWeight(600)
		}
		if !row.folder {
			ui.Text(c, formatBytes(row.size)).FontSize(11).TextColor(a.pal.faint).
				SingleLine().Margin(-4, 0, 0, 0)
		}
	})
	tile.ContextMenu(func(m *ui.Menu) { a.entryMenu(c, m, row) })
	if tile.DoubleClicked() {
		a.openEntry(row)
	}
}

// gridThumb is the picture of a tile: a folder, a striped image or the
// kind's icon.
func (a *App) gridThumb(c *ui.Context, row entry, selected bool) {
	kind := row.kind()
	kinds := a.pal.kindPaletteOf(kind)
	box := ui.Box(c).Size(96, 72).Radius(9).Center()
	switch {
	case row.folder:
	case kind == "image":
		box.Background(a.pal.tile).Stripes(a.pal.stripe, 6, 6, 135)
	default:
		box.Background(kinds.background)
	}
	if selected {
		box.Shadow(0, 0, 0, 2.5, a.pal.primary)
	} else if !row.folder {
		box.Border(1, a.pal.border)
	}
	box.Children(func() {
		switch {
		case row.folder:
			ui.Box(c).Size(58, 58).Children(func() {
				ui.Icon(c, folderFill).FontSize(58).TextColor(a.pal.folderFill).
					Absolute().Left(0).Top(0)
				ui.Icon(c, folderOutline).FontSize(58).TextColor(a.pal.folderStroke).
					Absolute().Left(0).Top(0)
			})
		case kind == "image":
		default:
			ui.Icon(c, iconOf(kind)).FontSize(30).TextColor(kinds.foreground)
		}
	})
}

// columnsView shows one list per step of the path, as Finder's column view.
func (a *App) columnsView(c *ui.Context) {
	location := a.location
	if location == nil {
		return
	}
	prefixes := []string{""}
	parts := strings.FieldsFunc(location.Prefix, func(r rune) bool { return r == '/' })
	for i := range parts {
		prefixes = append(prefixes, strings.Join(parts[:i+1], "/")+"/")
	}
	if a.browse.columnLists == nil {
		a.browse.columnLists = map[string]*ui.ListState{}
	}
	for _, prefix := range prefixes {
		a.loadColumn(Location{ProfileID: location.ProfileID, Bucket: location.Bucket, Prefix: prefix})
	}

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.ScrollHorizontal(c).Grow(1).FillHeight().Children(func() {
			ui.Row(c).FillHeight().AlignItems(ui.Stretch).Children(func() {
				for _, prefix := range prefixes {
					a.column(c, Location{
						ProfileID: location.ProfileID, Bucket: location.Bucket, Prefix: prefix,
					})
				}
			})
		})
		a.columnsPreview(c)
	})
}

// column draws one column of the columns view.
func (a *App) column(c *ui.Context, location Location) {
	state, ok := a.browse.columnLists[location.Prefix]
	if !ok {
		state = &ui.ListState{}
		a.browse.columnLists[location.Prefix] = state
	}
	rows := a.browse.columnRows[location.Prefix]
	busy := a.browse.columnBusy[location.Prefix]
	err := a.browse.columnErr[location.Prefix]

	ui.Column(c).Width(230).FillHeight().
		BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border).Children(func() {
		switch {
		case busy && rows == nil:
			ui.Column(c).Grow(1).Center().Children(func() { ui.Spinner(c) })
		case err != nil:
			ui.Column(c).Grow(1).Center().Padding(12).Children(func() {
				ui.Text(c, a.errorText(err)).FontSize(11.5).TextColor(a.pal.muted).TextAlign(ui.Center)
			})
		default:
			state.Key = func(row int) any { return rows[row].key }
			state.Label = func(row int) string { return rows[row].name }
			state.Selection = &a.browse.selection
			state.MenuKeepsSelection = false
			ui.List(c, state, len(rows), func(i int) {
				a.columnRow(c, rows[i])
			}).Fill().Padding(6).Gap(1)
		}
	})
}

// columnRow is one item of a column.
func (a *App) columnRow(c *ui.Context, row entry) {
	selected := a.browse.selection.Has(row.key) || a.onPath(row)
	kinds := a.pal.kindPaletteOf(row.kind())
	text, tileBg, tileFg := a.pal.text, kinds.background, kinds.foreground
	if selected {
		text, tileBg, tileFg = a.pal.onPrimary, ui.RGBA(255, 255, 255, 0.22), a.pal.onPrimary
	}
	item := ui.Row(c).Height(30).PaddingX(8).Gap(9).Radius(7).AlignItems(ui.Center)
	if selected {
		item.Background(a.pal.primary)
	}
	item.Children(func() {
		ui.Box(c).Size(22, 22).Radius(6).Center().Background(tileBg).Children(func() {
			ui.Icon(c, iconOf(row.kind())).FontSize(13).TextColor(tileFg)
		})
		name := ui.Text(c, row.name).FontSize(13).SingleLine().Grow(1).TextColor(text)
		if selected {
			name.FontWeight(500)
		}
		if row.folder {
			ui.Icon(c, iconChevronRight).FontSize(10).TextColor(text).Opacity(0.55)
		}
	})
	item.ContextMenu(func(m *ui.Menu) { a.entryMenu(c, m, row) })
	if item.Clicked() && row.folder && a.location != nil {
		a.navigate(Location{ProfileID: a.location.ProfileID, Bucket: a.location.Bucket, Prefix: row.key})
	}
	if item.DoubleClicked() {
		a.openEntry(row)
	}
}

// onPath reports whether a folder is one of the steps to the folder in view,
// which the columns view keeps marked.
func (a *App) onPath(row entry) bool {
	if !row.folder || a.location == nil {
		return false
	}
	return strings.HasPrefix(a.location.Prefix, row.key)
}

// columnsPreview shows the object chosen in the columns view.
func (a *App) columnsPreview(c *ui.Context) {
	chosen := a.selectedEntries()
	if len(chosen) != 1 || chosen[0].folder {
		return
	}
	row := chosen[0]
	kinds := a.pal.kindPaletteOf(row.kind())
	ui.Column(c).Width(300).FillHeight().Padding(22).Gap(14).Children(func() {
		box := ui.Box(c).Height(180).Radius(12).Center().Background(kinds.background).
			Border(1, a.pal.border)
		if row.kind() == "image" {
			box.Background(a.pal.tile).Stripes(a.pal.stripe, 6, 6, 135)
		}
		box.Children(func() {
			if row.kind() != "image" {
				ui.Icon(c, iconOf(row.kind())).FontSize(40).TextColor(kinds.foreground)
			}
		})
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, row.name).FontSize(14).FontWeight(600).MaxLines(2)
			ui.Text(c, a.t("kinds."+row.kind())+" · "+formatBytes(row.size)).FontSize(12).
				TextColor(a.pal.muted)
		})
		ui.Column(c).Gap(6).Children(func() {
			a.detailRow(c, a.t("browser.previewModified"), formatDate(row.modified))
			a.detailRow(c, a.t("browser.previewKey"), row.key)
		})
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, a.t("menu.quickLook")).Clicked() {
				a.openPreview(row.key)
			}
			if ui.Button(c, a.t("common.download")).Clicked() {
				a.download([]entry{row})
			}
		})
	})
}

// detailRow is a label and its value, as the preview shows them.
func (a *App) detailRow(c *ui.Context, label, value string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(a.pal.muted).Width(72)
		ui.Text(c, value).FontSize(12).SingleLine().Grow(1)
	})
}
