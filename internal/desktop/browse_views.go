package desktop

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// syncSort reads the order the user asked for from the table header.
func (a *App) syncSort() {
	order := a.browse.sort
	if order.Column == "" {
		return
	}
	descending := a.browse.sortDir < 0
	if order.Column == a.browse.sortKey && order.Descending == descending {
		return
	}
	a.browse.sortKey = order.Column
	a.browse.sortDir = 1
	if order.Descending {
		a.browse.sortDir = -1
	}
	a.prefs.SortKey, a.prefs.SortDir = a.browse.sortKey, a.browse.sortDir
	a.savePrefs()
	a.rebuildEntries()
}

// listView shows the folder as a table of columns.
func (a *App) listView(c *ui.Context) {
	a.syncSort()
	state := &a.browse
	state.list.Key = func(row int) any { return state.shown[row].key }
	state.list.Label = func(row int) string { return state.shown[row].name }
	state.list.Selection = &state.selection
	state.list.Selected = &state.selectedIndex
	state.list.Sort = &state.sort
	state.list.MenuKeepsSelection = false
	state.list.CellMenu = func(m *ui.Menu, row, col int) {
		if row < 0 || row >= len(state.shown) {
			return
		}
		a.entryMenu(c, m, state.shown[row])
	}
	state.list.Columns = ui.TableLayout{
		Order:  a.prefs.ColumnOrder,
		Widths: a.prefs.ColumnWidths,
	}

	columns := []ui.TableColumn{
		{Title: a.t("browser.colName"), ID: "name", Sortable: true, MinWidth: 160},
		{Title: a.t("browser.colSize"), ID: "size", Width: 96, Align: ui.End, Sortable: true},
		{Title: a.t("browser.colModified"), ID: "modified", Width: 150, Sortable: true},
		{Title: a.t("browser.colKind"), ID: "kind", Width: 92},
	}
	table := ui.Table(c, &state.list, columns, len(state.shown), func(row, col int) {
		a.tableCell(c, state.shown[row], col)
	}).Fill().Label(a.t("browser.colName"))
	if table.Submitted() {
		if state.selectedIndex >= 0 && state.selectedIndex < len(state.shown) {
			a.openEntry(state.shown[state.selectedIndex])
		}
	}
	a.rememberColumns(state.list.Columns)
}

// rememberColumns saves the column order and widths the user gave the list.
func (a *App) rememberColumns(layout ui.TableLayout) {
	if len(layout.Order) == 0 && len(layout.Widths) == 0 {
		return
	}
	if sameStrings(layout.Order, a.prefs.ColumnOrder) && sameWidths(layout.Widths, a.prefs.ColumnWidths) {
		return
	}
	a.prefs.ColumnOrder = append([]string{}, layout.Order...)
	widths := make(map[string]float32, len(layout.Widths))
	for name, width := range layout.Widths {
		widths[name] = width
	}
	a.prefs.ColumnWidths = widths
	a.savePrefs()
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameWidths(a, b map[string]float32) bool {
	if len(a) != len(b) {
		return false
	}
	for name, width := range a {
		if b[name] != width {
			return false
		}
	}
	return true
}

// tableCell builds one cell of the list view.
func (a *App) tableCell(c *ui.Context, row entry, col int) {
	selected := a.browse.selection.Has(row.key)
	muted := a.pal.muted
	if selected {
		muted = a.pal.onPrimary
	}
	switch col {
	case 0:
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconOf(row.kind())).FontSize(16).
				TextColor(a.pal.kindPaletteOf(row.kind()).foreground)
			ui.Text(c, row.name).FontSize(13).SingleLine().TextColor(a.cellText(selected))
		})
	case 1:
		text := "—"
		if !row.folder {
			text = formatBytes(row.size)
		}
		ui.Text(c, text).FontSize(12.5).TextAlign(ui.End).SingleLine().TextColor(muted)
	case 2:
		text := "—"
		if !row.folder {
			text = formatDate(row.modified)
		}
		ui.Text(c, text).FontSize(12.5).SingleLine().TextColor(muted)
	default:
		ui.Text(c, a.t("kinds."+row.kind())).FontSize(11.5).SingleLine().TextColor(muted)
	}
}

// cellText is the color of a cell's main text.
func (a *App) cellText(selected bool) ui.Color {
	if selected {
		return a.pal.onPrimary
	}
	return a.pal.text
}

// gridView shows the folder as a grid of tiles.
func (a *App) gridView(c *ui.Context) {
	state := &a.browse
	state.grid.Key = func(item int) any { return state.shown[item].key }
	state.grid.Label = func(item int) string { return state.shown[item].name }
	state.grid.Selection = &state.selection
	state.grid.Selected = &state.selectedIndex

	grid := ui.GridView(c, &state.grid, len(state.shown), 118, 126, func(i int) {
		a.gridTile(c, state.shown[i])
	}).Fill()
	if grid.Submitted() {
		if state.selectedIndex >= 0 && state.selectedIndex < len(state.shown) {
			a.openEntry(state.shown[state.selectedIndex])
		}
	}
}

// gridTile draws one tile of the grid view.
func (a *App) gridTile(c *ui.Context, row entry) {
	selected := a.browse.selection.Has(row.key)
	tile := ui.Column(c).Gap(6).Padding(6).Radius(12).AlignItems(ui.Center)
	if selected {
		tile.Background(a.pal.primaryTint).Border(2, a.pal.primary)
	} else if tile.Hovered() {
		tile.Background(a.pal.hover)
	}
	tile.Children(func() {
		kinds := a.pal.kindPaletteOf(row.kind())
		box := ui.Box(c).Size(96, 68).Radius(10).Center().Background(kinds.background)
		if row.kind() == "image" {
			box.Stripes(a.pal.mutedFill, 5, 5, 135)
		}
		box.Children(func() {
			ui.Icon(c, iconOf(row.kind())).FontSize(30).TextColor(kinds.foreground)
		})
		color := a.pal.text
		if selected {
			color = a.pal.primaryText
		}
		ui.Text(c, row.name).FontSize(11.5).TextAlign(ui.Center).MaxLines(2).TextColor(color)
	})
	tile.ContextMenu(func(m *ui.Menu) { a.entryMenu(c, m, row) })
	if tile.DoubleClicked() {
		a.openEntry(row)
	}
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

	header := location.Bucket
	if location.Prefix != "" {
		header = basename(location.Prefix)
	}

	ui.Column(c).Width(232).FillHeight().
		BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border).Children(func() {
		ui.Row(c).Height(30).PaddingX(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, header).FontSize(11).Bold().TextColor(a.pal.faint).SingleLine()
		})
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
			list := ui.List(c, state, len(rows), func(i int) {
				row := rows[i]
				selected := a.browse.selection.Has(row.key)
				item := ui.Row(c).Padding(6, 10).Gap(8).AlignItems(ui.Center)
				if selected {
					item.Background(a.pal.primary).Radius(6)
				}
				item.Children(func() {
					ui.Icon(c, iconOf(row.kind())).FontSize(15).
						TextColor(a.pal.kindPaletteOf(row.kind()).foreground)
					ui.Text(c, row.name).FontSize(12.5).SingleLine().Grow(1).
						TextColor(a.cellText(selected))
					if row.folder {
						ui.Icon(c, iconChevronRight).FontSize(13).TextColor(a.pal.faint)
					}
				})
				item.ContextMenu(func(m *ui.Menu) { a.entryMenu(c, m, row) })
				if item.DoubleClicked() {
					a.openEntry(row)
				}
			}).Fill()
			_ = list
		}
	})
}

// columnsPreview shows the object chosen in the columns view.
func (a *App) columnsPreview(c *ui.Context) {
	chosen := a.selectedEntries()
	if len(chosen) != 1 || chosen[0].folder {
		return
	}
	row := chosen[0]
	kinds := a.pal.kindPaletteOf(row.kind())
	ui.Column(c).Width(280).FillHeight().Padding(16).Gap(8).
		BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border).
		Background(a.pal.sidebar).Children(func() {
		ui.Box(c).Height(120).Radius(10).Center().Background(kinds.background).Children(func() {
			ui.Icon(c, iconOf(row.kind())).FontSize(36).TextColor(kinds.foreground)
		})
		ui.Text(c, row.name).FontSize(13).Bold().MaxLines(2)
		a.detailRow(c, a.t("browser.previewSize"), formatBytes(row.size))
		a.detailRow(c, a.t("browser.previewModified"), formatDate(row.modified))
		a.detailRow(c, a.t("browser.colKind"), a.t("kinds."+row.kind()))
		ui.Spacer(c)
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
		ui.Text(c, label).FontSize(11.5).TextColor(a.pal.faint).Width(88)
		ui.Text(c, value).FontSize(11.5).TextColor(a.pal.muted).SingleLine().Grow(1)
	})
}
