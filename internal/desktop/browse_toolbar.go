package desktop

import "github.com/egoist/mygo/ui"

// browserToolbar is the row of controls above the file list.
func (a *App) browserToolbar(c *ui.Context) {
	ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Center).Children(func() {
		a.toolButton(c, iconChevronLeft, a.t("browser.back"), len(a.backStack) == 0, a.goBack)
		a.toolButton(c, iconChevronRight, a.t("browser.forward"), len(a.forwardStack) == 0, a.goForward)
		a.breadcrumbBar(c)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			a.searchField(c)
			a.viewSwitch(c)
			a.sortButton(c)
			a.uploadButton(c)
		})
	})
}

// toolButton is a small icon-only button of the toolbar.
func (a *App) toolButton(c *ui.Context, glyph *ui.SVG, label string, disabled bool, action func()) {
	button := a.plainButton(c, label, 28, 28).Disabled(disabled)
	color := a.pal.muted
	if disabled {
		color = a.pal.faint
		button.Background(ui.Transparent)
	}
	button.Children(func() { ui.Icon(c, glyph).FontSize(16).TextColor(color) })
	if !disabled && button.Clicked() {
		action()
	}
}

// breadcrumbBar shows the connection, the bucket and the path to the folder
// in view, then how many items it holds.
func (a *App) breadcrumbBar(c *ui.Context) {
	if a.location == nil {
		return
	}
	location := *a.location
	profile := a.currentProfile()
	crumbs := breadcrumbs(location.Prefix)
	ui.Row(c).Grow(1).MinWidth(0).Gap(1).MarginStart(6).AlignItems(ui.Center).Clip().Children(func() {
		if profile != nil {
			a.crumb(c, profile.Name, false, func() {
				ui.Box(c).Size(16, 16).Children(func() { a.providerTile(c, profile.ProviderType, 16) })
			}, func() {
				a.navigate(Location{ProfileID: location.ProfileID, Bucket: location.Bucket})
			})
			a.crumbSeparator(c)
		}
		a.crumb(c, location.Bucket, len(crumbs) == 0, nil, func() {
			a.navigate(Location{ProfileID: location.ProfileID, Bucket: location.Bucket})
		})
		for i, step := range crumbs {
			prefix := step.prefix
			a.crumbSeparator(c)
			a.crumb(c, step.name, i == len(crumbs)-1, nil, func() {
				a.navigate(Location{ProfileID: location.ProfileID, Bucket: location.Bucket, Prefix: prefix})
			})
		}
		count := a.tn("browser.items", len(a.browse.shown), "count", formatCount(int64(len(a.browse.shown))))
		ui.Text(c, count).FontSize(12).TextColor(a.pal.faint).SingleLine().MarginStart(8)
	})
}

// crumb is one step of the path; the last is the folder in view.
func (a *App) crumb(c *ui.Context, label string, last bool, lead func(), action func()) {
	item := ui.Row(c).Height(26).PaddingX(8).Gap(7).Radius(6).AlignItems(ui.Center).
		Shrink(0).Label(label)
	if !last && item.Hovered() {
		item.Background(a.pal.hover)
	}
	item.Children(func() {
		if lead != nil {
			lead()
		}
		text := ui.Text(c, label).FontSize(13).SingleLine()
		if last {
			text.FontWeight(600).TextColor(a.pal.text)
		} else {
			text.TextColor(a.pal.muted)
		}
	})
	if !last && item.Clicked() {
		action()
	}
}

func (a *App) crumbSeparator(c *ui.Context) {
	ui.Icon(c, iconChevronRight).FontSize(12).TextColor(a.pal.faint)
}

// searchField searches inside the folder in view. It folds to its icon
// until it is used.
func (a *App) searchField(c *ui.Context) {
	state := &a.browse.search
	if !a.browse.searchOpen && state.query == "" {
		label := a.t("browser.searchTip")
		button := a.plainButton(c, label, 28, 28)
		button.Children(func() { ui.Icon(c, iconSearch).FontSize(16).TextColor(a.pal.muted) })
		if button.Clicked() {
			a.browse.searchOpen = true
		}
		return
	}
	field := ui.SearchField(c, &state.query).Label(a.t("browser.searchPlaceholder")).
		Width(220).Height(28).AutoFocus()
	if field.Changed() || field.Submitted() {
		a.setQuery(state.query)
	}
	if state.query == "" && !field.FocusWithin() {
		a.browse.searchOpen = false
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
	const width = 32 + 2*segmentInset
	seg.Track.Size(width*float32(len(views)), segmentHeight).Radius(8).Background(a.pal.mutedFill).Children(func() {
		a.segmentThumb(c).Left(width * float32(index)).Width(width)
		glyphs := []*ui.SVG{iconList, iconLayoutGrid, iconColumns}
		labels := []string{a.t("browser.list"), a.t("browser.grid"), a.t("browser.columns")}
		for i, glyph := range glyphs {
			segment := seg.Segment(i).Absolute().Left(width*float32(i)).Top(0).Size(width, segmentHeight).Center().
				Label(labels[i]).Tooltip(labels[i])
			color := a.pal.muted
			if i == index {
				color = a.pal.text
			}
			segment.Children(func() { ui.Icon(c, glyph).FontSize(15).TextColor(color) })
		}
	})
	if index >= 0 && index < len(views) && views[index] != a.browse.view {
		a.setView(views[index])
	}
}

// sortButton chooses the column and the direction rows are sorted by.
func (a *App) sortButton(c *ui.Context) {
	label := a.t("browser.sort")
	button := a.plainButton(c, label, 30, 28)
	button.Children(func() { ui.Icon(c, iconArrowDownUp).FontSize(16).TextColor(a.pal.muted) })
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
		if m.Item(a.t("browser.ascending")).Checked(a.browse.sortDir > 0).Chosen() {
			a.setSort(a.browse.sortKey, 1)
		}
		if m.Item(a.t("browser.descending")).Checked(a.browse.sortDir < 0).Chosen() {
			a.setSort(a.browse.sortKey, -1)
		}
	})
}

// setSort orders the rows by a column and a direction.
func (a *App) setSort(key string, direction int) {
	if direction == 0 {
		direction = 1
	}
	a.browse.sortKey, a.browse.sortDir = key, direction
	a.prefs.SortKey, a.prefs.SortDir = key, direction
	a.savePrefs()
	a.rebuildEntries()
}

// uploadButton is the page's one primary action: upload files or a folder.
func (a *App) uploadButton(c *ui.Context) {
	readOnly := a.readOnly()
	button := a.primaryAction(c, iconUpload, a.t("browser.upload"), readOnly)
	if readOnly {
		button.Tooltip(a.t("browser.readOnlyTip"))
		return
	}
	button.Menu(func(m *ui.Menu) {
		if m.Item(a.t("browser.uploadFiles")).Chosen() {
			a.pickAndUpload(false)
		}
		if m.Item(a.t("browser.uploadFolder")).Chosen() {
			a.pickAndUpload(true)
		}
	})
}
