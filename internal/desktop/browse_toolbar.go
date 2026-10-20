package desktop

import "github.com/egoist/mygo/ui"

// browserToolbar is the row of controls above the file list.
func (a *App) browserToolbar(c *ui.Context) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		a.toolButton(c, iconChevronLeft, a.t("browser.back"), len(a.backStack) == 0, a.goBack)
		a.toolButton(c, iconChevronRight, a.t("browser.forward"), len(a.forwardStack) == 0, a.goForward)
		a.breadcrumbBar(c)
		ui.Spacer(c)
		a.searchField(c)
		a.viewSwitch(c)
		a.sortButton(c)
		a.uploadButton(c)
	})
}

// toolButton is a small icon-only button of the toolbar.
func (a *App) toolButton(c *ui.Context, glyph *ui.SVG, label string, disabled bool, action func()) {
	if disabled {
		button := ui.Button(c, "").Label(label).Tooltip(label).Size(28, 28).Disabled(true)
		button.Background(ui.Transparent)
		button.Children(func() { ui.Icon(c, glyph).FontSize(16) })
		return
	}
	a.ghostButton(c, glyph, label, 28, false, action)
}

// breadcrumbBar shows the path to the folder in view.
func (a *App) breadcrumbBar(c *ui.Context) {
	if a.location == nil {
		return
	}
	location := *a.location
	crumbs := breadcrumbs(location.Prefix)
	items := make([]string, 0, len(crumbs)+1)
	items = append(items, location.Bucket)
	for _, step := range crumbs {
		items = append(items, step.name)
	}
	chosen := -1
	bar := ui.Breadcrumbs(c, items, &chosen).Label(a.t("common.bucket"))
	bar.Children(func() {
		ui.Icon(c, iconDatabase).FontSize(14).TextColor(a.pal.muted)
	})
	if bar.Changed() && chosen >= 0 {
		prefix := ""
		if chosen > 0 && chosen-1 < len(crumbs) {
			prefix = crumbs[chosen-1].prefix
		}
		a.navigate(Location{ProfileID: location.ProfileID, Bucket: location.Bucket, Prefix: prefix})
	}
}

// searchField starts and clears the search inside the folder in view.
func (a *App) searchField(c *ui.Context) {
	state := &a.browse.search
	field := ui.SearchField(c, &state.query).Label(a.t("browser.searchPlaceholder")).Width(220)
	if field.Changed() {
		a.setQuery(state.query)
	}
	if field.Submitted() {
		a.setQuery(state.query)
	}
}

// viewSwitch chooses between the list, the grid and the columns.
func (a *App) viewSwitch(c *ui.Context) {
	views := []string{viewList, viewGrid, viewColumns}
	index := 0
	for i, name := range views {
		if name == a.browse.view {
			index = i
		}
	}
	seg := ui.SegmentedBase(c, &index, len(views))
	seg.Track.Padding(2).Radius(8).Background(a.pal.field).Children(func() {
		glyphs := []*ui.SVG{iconList, iconLayoutGrid, iconColumns}
		labels := []string{a.t("browser.list"), a.t("browser.grid"), a.t("browser.columns")}
		for i, glyph := range glyphs {
			segment := seg.Segment(i).Padding(4, 8).Radius(6).Label(labels[i]).Tooltip(labels[i])
			if i == index {
				segment.Background(a.pal.background)
			}
			segment.Children(func() { ui.Icon(c, glyph).FontSize(15) })
		}
	})
	if index >= 0 && index < len(views) && views[index] != a.browse.view {
		a.browse.view = views[index]
		a.prefs.View = a.browse.view
		a.savePrefs()
	}
}

// sortButton chooses the column and the direction rows are sorted by.
func (a *App) sortButton(c *ui.Context) {
	button := ui.Button(c, "").Label(a.t("browser.sort")).Tooltip(a.t("browser.sort")).Size(28, 28)
	button.Children(func() { ui.Icon(c, iconArrowDownUp).FontSize(16) })
	button.Menu(func(m *ui.Menu) {
		for _, option := range []struct{ key, label string }{
			{"name", a.t("browser.sortName")},
			{"size", a.t("browser.sortSize")},
			{"modified", a.t("browser.sortModified")},
		} {
			if m.Item(option.label).Checked(a.browse.sortKey == option.key).Chosen() {
				a.setSort(option.key, a.browse.sortDir)
			}
		}
		m.Separator()
		direction := a.t("browser.ascending")
		if a.browse.sortDir < 0 {
			direction = a.t("browser.descending")
		}
		if m.Item(direction).Chosen() {
			a.setSort(a.browse.sortKey, -a.browse.sortDir)
		}
	})
}

// setSort orders the rows by a column and a direction.
func (a *App) setSort(key string, direction int) {
	if direction == 0 {
		direction = 1
	}
	a.browse.sortKey, a.browse.sortDir = key, direction
	a.browse.sort = ui.SortOrder{Column: key, Descending: direction < 0}
	a.prefs.SortKey, a.prefs.SortDir = key, direction
	a.savePrefs()
	a.rebuildEntries()
}

// uploadButton uploads files or a folder, or opens the image host.
func (a *App) uploadButton(c *ui.Context) {
	label := a.t("browser.upload")
	if a.readOnly() {
		button := ui.Button(c, label).Disabled(true).Tooltip(a.t("browser.readOnlyTip"))
		button.Children(func() { ui.Icon(c, iconUpload).FontSize(14) })
		return
	}
	button := ui.Button(c, "").Label(label).Tooltip(a.t("browser.upload"))
	button.Children(func() {
		ui.Icon(c, iconUpload).FontSize(14)
		ui.Text(c, label).FontSize(12.5)
		ui.Icon(c, iconChevronDown).FontSize(13).TextColor(a.pal.muted)
	})
	button.Menu(func(m *ui.Menu) {
		if m.Item(a.t("browser.uploadFiles")).Chosen() {
			a.pickAndUpload(false)
		}
		if m.Item(a.t("browser.uploadFolder")).Chosen() {
			a.pickAndUpload(true)
		}
		m.Separator()
		if m.Item(a.t("quick.title")).Chosen() {
			a.openQuick()
		}
	})
}
