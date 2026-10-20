package desktop

import (
	"context"
	"strings"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// pageSize is how many objects one listing request asks for.
const pageSize = 1000

// loadProfiles reads the connections.
func (a *App) loadProfiles() {
	a.profilesBusy = true
	go func() {
		profiles, err := a.svc.Profiles(a.ctx)
		a.update(func() {
			a.profilesBusy = false
			a.profilesErr = err
			if err == nil {
				a.profiles = profiles
			}
			a.pruneProfiles()
		})
	}()
}

// pruneProfiles drops connections that no longer exist from the history.
func (a *App) pruneProfiles() {
	known := map[string]bool{}
	for _, profile := range a.profiles {
		known[profile.ID] = true
	}
	for _, location := range append(append([]Location{}, a.backStack...), a.forwardStack...) {
		if !known[location.ProfileID] {
			a.forgetProfile(location.ProfileID)
			break
		}
	}
	if a.location != nil && !known[a.location.ProfileID] {
		a.forgetProfile(a.location.ProfileID)
	}
}

// ensureBuckets reads the buckets of a connection once.
func (a *App) ensureBuckets(profileID string) {
	if profileID == "" {
		return
	}
	state, ok := a.buckets[profileID]
	if ok && (state.loaded || state.busy) {
		return
	}
	a.buckets[profileID] = &bucketState{busy: true}
	go func() {
		buckets, err := a.svc.Buckets(a.ctx, profileID)
		a.update(func() {
			a.buckets[profileID] = &bucketState{buckets: buckets, err: err, loaded: true}
		})
	}()
}

// loadBuckets re-reads the buckets of a connection.
func (a *App) loadBuckets(profileID string) {
	if profileID == "" {
		return
	}
	a.buckets[profileID] = &bucketState{busy: true}
	go func() {
		buckets, err := a.svc.Buckets(a.ctx, profileID)
		a.update(func() {
			a.buckets[profileID] = &bucketState{buckets: buckets, err: err, loaded: true}
		})
	}()
}

// loadEntries reads the first page of the folder in view.
func (a *App) loadEntries() {
	location := a.location
	if location == nil {
		return
	}
	a.browse.loading = true
	a.browse.err = nil
	go func() {
		page, err := a.svc.List(a.ctx, core.ListRequest{
			ProfileID: location.ProfileID, Bucket: location.Bucket,
			Prefix: location.Prefix, Delimiter: "/", Limit: pageSize,
		})
		a.update(func() {
			if !sameLocation(a.location, location) {
				return
			}
			a.browse.loading = false
			if err != nil {
				a.browse.err = err
				return
			}
			a.browse.pages = []core.ListPage{page}
			a.browse.cursor = page.NextCursor
			a.browse.more = page.NextCursor != ""
			a.rebuildEntries()
		})
	}()
}

// loadMore reads the next page of the folder in view.
func (a *App) loadMore() {
	location := a.location
	if location == nil || a.browse.moreBusy || !a.browse.more || a.browse.search.active {
		return
	}
	cursor := a.browse.cursor
	a.browse.moreBusy = true
	go func() {
		page, err := a.svc.List(a.ctx, core.ListRequest{
			ProfileID: location.ProfileID, Bucket: location.Bucket,
			Prefix: location.Prefix, Delimiter: "/", Cursor: cursor, Limit: pageSize,
		})
		a.update(func() {
			if !sameLocation(a.location, location) {
				return
			}
			a.browse.moreBusy = false
			if err != nil {
				a.browse.err = err
				return
			}
			a.browse.pages = append(a.browse.pages, page)
			a.browse.cursor = page.NextCursor
			a.browse.more = page.NextCursor != ""
			a.rebuildEntries()
		})
	}()
}

// loadColumn reads one column of the columns view.
func (a *App) loadColumn(location Location) {
	if a.browse.columnRows == nil {
		a.browse.columnRows = map[string][]entry{}
		a.browse.columnBusy = map[string]bool{}
		a.browse.columnErr = map[string]error{}
	}
	if _, done := a.browse.columnRows[location.Prefix]; done {
		return
	}
	if a.browse.columnBusy[location.Prefix] {
		return
	}
	a.browse.columnBusy[location.Prefix] = true
	go func() {
		pages := []core.ListPage{}
		cursor := ""
		for {
			page, err := a.svc.List(a.ctx, core.ListRequest{
				ProfileID: location.ProfileID, Bucket: location.Bucket,
				Prefix: location.Prefix, Delimiter: "/", Cursor: cursor, Limit: pageSize,
			})
			if err != nil {
				a.update(func() {
					a.browse.columnBusy[location.Prefix] = false
					a.browse.columnErr[location.Prefix] = err
				})
				return
			}
			pages = append(pages, page)
			if page.NextCursor == "" || len(pages) >= 4 {
				break
			}
			cursor = page.NextCursor
		}
		rows := sortEntries(entriesFromPages(pages, location.Prefix), "name", 1)
		a.update(func() {
			a.browse.columnBusy[location.Prefix] = false
			a.browse.columnErr[location.Prefix] = nil
			a.browse.columnRows[location.Prefix] = rows
		})
	}()
}

// rebuildEntries sorts the loaded rows and refreshes what depends on them.
func (a *App) rebuildEntries() {
	if a.browse.search.active {
		a.browse.shown = sortEntries(entriesFromFind(a.browse.search.objects, a.prefix()), a.browse.sortKey, a.browse.sortDir)
	} else {
		rows := entriesFromPages(a.browse.pages, a.prefix())
		a.browse.rows = rows
		a.browse.shown = sortEntries(rows, a.browse.sortKey, a.browse.sortDir)
	}
}

func (a *App) prefix() string {
	if a.location == nil {
		return ""
	}
	return a.location.Prefix
}

// setQuery starts or clears the search inside the folder in view.
func (a *App) setQuery(query string) {
	a.browse.search.query = query
	if strings.TrimSpace(query) == "" {
		a.browse.search.active = false
		a.browse.search.objects = nil
		a.browse.search.err = nil
		a.browse.search.busy = false
		a.browse.search.complete = false
		a.browse.selection.Clear()
		a.rebuildEntries()
		return
	}
	a.browse.search.active = true
	a.browse.search.objects = nil
	a.browse.search.scanned = 0
	a.browse.search.cursor = ""
	a.browse.search.complete = false
	a.browse.search.err = nil
	a.browse.search.dirty = true
	a.browse.selection.Clear()
	a.rebuildEntries()
	a.startPendingSearch()
}

// startPendingSearch runs the search for the query typed since the last
// scan, and does nothing while one is already running: the keystrokes of a
// fast typist then cost one request, not one each.
func (a *App) startPendingSearch() {
	state := &a.browse.search
	if !state.active || state.busy || !state.dirty {
		return
	}
	state.dirty = false
	a.runSearch("")
}

// runSearch reads a page of search results.
func (a *App) runSearch(cursor string) {
	location := a.location
	if location == nil {
		return
	}
	query := strings.TrimSpace(a.browse.search.query)
	if query == "" {
		return
	}
	a.browse.search.busy = true
	go func() {
		result, err := a.svc.Find(a.ctx, core.FindRequest{
			ProfileID: location.ProfileID, Bucket: location.Bucket, Prefix: location.Prefix,
			Contains: query, Limit: pageSize, Cursor: cursor,
		})
		a.update(func() {
			if !sameLocation(a.location, location) || !a.browse.search.active {
				return
			}
			// The query may have changed while the scan ran: keep the
			// results only when it has not.
			if strings.TrimSpace(a.browse.search.query) != query {
				a.browse.search.busy = false
				a.startPendingSearch()
				return
			}
			a.browse.search.busy = false
			if err != nil {
				a.browse.search.err = err
				a.startPendingSearch()
				return
			}
			if cursor == "" {
				a.browse.search.objects = result.Objects
			} else {
				a.browse.search.objects = append(a.browse.search.objects, result.Objects...)
			}
			a.browse.search.scanned += result.Scanned
			a.browse.search.complete = result.Complete
			a.browse.search.cursor = result.NextCursor
			a.browse.search.more = !result.Complete && result.NextCursor != ""
			a.rebuildEntries()
			a.startPendingSearch()
		})
	}()
}

// selectedEntries are the rows the user chose, in the order shown.
func (a *App) selectedEntries() []entry {
	out := []entry{}
	for _, row := range a.browse.shown {
		if a.browse.selection.Has(row.key) {
			out = append(out, row)
		}
	}
	return out
}

// clearSelection takes the choice away.
func (a *App) clearSelection() {
	a.browse.selection.Clear()
	a.browse.focus = ""
}

// openEntry follows a folder, or previews an object.
func (a *App) openEntry(row entry) {
	if a.location == nil {
		return
	}
	if row.folder {
		a.navigate(Location{ProfileID: a.location.ProfileID, Bucket: a.location.Bucket, Prefix: row.key})
		return
	}
	a.openPreview(row.key)
}

// browser draws the file browser.
func (a *App) browser(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		if a.readOnly() {
			a.banner(c, iconLock, a.t("browser.readOnlyBanner"), a.pal.warningBg, a.pal.warningText)
		}
		if a.browse.search.active {
			a.searchBanner(c)
		}
		zone := ui.Column(c).Grow(1)
		dragging := zone.FileDragOver()
		zone.Children(func() { a.browserBody(c) })
		if files := zone.DroppedFiles(); len(files) > 0 {
			if a.readOnly() {
				a.toastCurrent(a.t("browser.dropReadOnly"))
			} else if a.location != nil {
				a.startUploads(*a.location, files)
			}
		}
		if dragging {
			a.dropOverlay(c)
		}
		a.selectionBar(c)
		a.browserFooter(c)
	})
}

// dropOverlay shows where files dropped on the window would go.
func (a *App) dropOverlay(c *ui.Context) {
	place := a.prefix()
	if place == "" && a.location != nil {
		place = a.location.Bucket
	}
	text := a.t("browser.dropReadOnly")
	if !a.readOnly() {
		text = a.tf("browser.dropRelease", "place", place)
	}
	ui.Overlay(c, func() {
		ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(a.pal.primaryTint).Center().PassThrough().Children(func() {
			box := ui.Column(c).Padding(28, 40).Radius(16).Gap(8).AlignItems(ui.Center).
				Background(a.pal.popover).Border(2, a.pal.primaryLine)
			box.BorderStyle(ui.BorderDashed)
			box.Children(func() {
				ui.Icon(c, iconUploadCloud).FontSize(28).TextColor(a.pal.primary)
				ui.Text(c, text).FontSize(13).Bold().TextAlign(ui.Center)
			})
		})
	})
}

// banner is a one-line notice under the toolbar.
func (a *App) banner(c *ui.Context, icon *ui.SVG, text string, background, color ui.Color) {
	ui.Row(c).Height(30).PaddingX(16).Gap(6).AlignItems(ui.Center).
		Background(background).BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(color)
		ui.Text(c, text).FontSize(11.5).TextColor(color).SingleLine().Grow(1)
	})
}

// searchBanner reports the scope and the progress of the search.
func (a *App) searchBanner(c *ui.Context) {
	a.startPendingSearch()
	place := a.prefix()
	if place == "" {
		place = a.location.Bucket
	}
	state := &a.browse.search
	ui.Row(c).Height(30).PaddingX(16).Gap(6).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		ui.Icon(c, iconSearch).FontSize(13).TextColor(a.pal.faint)
		ui.Text(c, a.tf("search.scope", "place", place)).FontSize(11.5).
			TextColor(a.pal.muted).SingleLine()
		ui.Text(c, "·").TextColor(a.pal.faint)
		label := a.tn("search.scanned", int(state.scanned), "count", formatCount(state.scanned))
		if state.busy {
			label = a.t("search.scanning")
		}
		ui.Text(c, label).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
		ui.Spacer(c)
		if state.more {
			button := ui.Button(c, a.t("search.scanMore")).FontSize(11.5)
			if button.Clicked() {
				a.runSearch(state.cursor)
			}
			return
		}
		ui.Text(c, a.t("search.complete")).FontSize(11.5).TextColor(a.pal.faint)
	})
}

// browserBody shows the loading, error, empty or listing state.
func (a *App) browserBody(c *ui.Context) {
	state := &a.browse
	visibleErr := state.err
	if state.search.active {
		visibleErr = state.search.err
	}
	switch {
	case state.loading || (state.search.active && state.search.busy && len(state.search.objects) == 0):
		a.loadingState(c)
	case visibleErr != nil:
		a.errorState(c, visibleErr)
	case len(state.shown) == 0:
		a.emptyState(c)
	default:
		switch a.browse.view {
		case viewGrid:
			a.gridView(c)
		case viewColumns:
			a.columnsView(c)
		default:
			a.listView(c)
		}
	}
}

// loadingState shows the spinner while a listing is on its way.
func (a *App) loadingState(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(10).Children(func() {
		ui.Spinner(c).Label(a.t("states.loading"))
		ui.Text(c, a.t("states.loading")).FontSize(12.5).TextColor(a.pal.muted)
	})
}

// emptyState shows what to do when a folder has nothing to show.
func (a *App) emptyState(c *ui.Context) {
	state := &a.browse
	ui.Column(c).Fill().Center().Padding(24).Children(func() {
		ui.Column(c).MaxWidth(360).Gap(8).AlignItems(ui.Center).Children(func() {
			if state.search.active {
				a.emptyIcon(c, iconSearchX, false)
				ui.Text(c, a.tf("states.noResultsTitle", "query", strings.TrimSpace(state.search.query))).
					FontSize(13.5).Bold().TextAlign(ui.Center)
				place := a.prefix()
				if place == "" {
					place = a.location.Bucket
				}
				ui.Text(c, a.tf("states.noResultsDesc", "place", place,
					"count", formatCount(state.search.scanned))).
					FontSize(12).TextColor(a.pal.muted).TextAlign(ui.Center)
				return
			}
			a.emptyIcon(c, iconFolder, true)
			title := a.t("states.emptyBucketTitle")
			description := a.t("states.emptyBucketDesc")
			if a.prefix() != "" {
				title = a.t("states.emptyFolderTitle")
				description = a.tf("states.emptyFolderDesc", "place", a.prefix())
			}
			if a.readOnly() {
				description = a.t("states.readOnlyEmpty")
			}
			ui.Text(c, title).FontSize(13.5).Bold().TextAlign(ui.Center)
			ui.Text(c, description).FontSize(12).TextColor(a.pal.muted).TextAlign(ui.Center)
			if !a.readOnly() {
				if ui.PrimaryButton(c, a.t("browser.upload")).Clicked() {
					a.pickAndUpload(false)
				}
			}
		})
	})
}

func (a *App) emptyIcon(c *ui.Context, glyph *ui.SVG, accent bool) {
	color := a.pal.kindPaletteOf("other").foreground
	background := a.pal.tile
	if accent {
		color = a.pal.kindPaletteOf("folder").foreground
		background = a.pal.kindPaletteOf("folder").background
	}
	ui.Box(c).Size(52, 52).Radius(14).Background(background).Center().Children(func() {
		ui.Icon(c, glyph).FontSize(24).TextColor(color)
	})
}

// errorState explains why a listing failed.
func (a *App) errorState(c *ui.Context, err error) {
	code := core.AsError(err).Code
	glyph := iconAlertTriangle
	tone := "bad"
	title := a.t("states.errorTitle")
	description := a.errorText(err)
	editable := false
	switch code {
	case "credentials_unavailable", "configuration":
		glyph, tone = iconKeyRound, "warn"
		title, description, editable = a.t("states.credTitle"), a.t("states.credDesc"), true
	case "forbidden":
		glyph, tone = iconShieldX, "warn"
		title, description = a.t("states.permTitle"), a.t("states.permDesc")
	case "network", "timeout", "protocol":
		glyph, tone = iconWifiOff, "plain"
		title, description = a.t("states.netTitle"), a.t("states.netDesc")
	}
	color := a.pal.destructive
	background := a.pal.destructiveBg
	switch tone {
	case "warn":
		color, background = a.pal.warningText, a.pal.warningBg
	case "plain":
		color, background = a.pal.muted, a.pal.tile
	}
	ui.Column(c).Fill().Center().Padding(24).Children(func() {
		ui.Column(c).MaxWidth(380).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(52, 52).Radius(14).Background(background).Center().Children(func() {
				ui.Icon(c, glyph).FontSize(24).TextColor(color)
			})
			ui.Text(c, title).FontSize(13.5).Bold().TextAlign(ui.Center)
			ui.Text(c, description).FontSize(12).TextColor(a.pal.muted).TextAlign(ui.Center)
			ui.Row(c).Gap(8).Children(func() {
				if editable {
					if ui.PrimaryButton(c, a.t("states.editConnection")).Clicked() {
						a.openConnect(a.location.ProfileID)
					}
				}
				if ui.Button(c, a.t("common.retry")).Clicked() {
					a.reload()
				}
			})
		})
	})
}

// reload reads the folder again, or runs the search again.
func (a *App) reload() {
	if a.browse.search.active {
		a.setQuery(a.browse.search.query)
		return
	}
	a.loadEntries()
}

// browserFooter reports how much is loaded and offers the next page. The
// toolbar counts a folder that is loaded whole, so it shows only for a
// search or a folder with more to load.
func (a *App) browserFooter(c *ui.Context) {
	state := &a.browse
	if !state.search.active && !state.more {
		return
	}
	ui.Row(c).Height(28).PaddingX(14).Gap(8).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		label := a.tn("browser.loaded", len(state.rows), "count", formatCount(int64(len(state.rows))))
		if state.search.active {
			label = a.tn("search.results", len(state.shown), "count", formatCount(int64(len(state.shown))))
		}
		ui.Text(c, label).FontSize(10.5).TextColor(a.pal.faint)
		if state.search.active && state.search.more {
			ui.Text(c, a.t("search.scanning")).FontSize(10.5).TextColor(a.pal.faint)
		}
		ui.Spacer(c)
		if !state.search.active && state.more {
			text := a.t("browser.loadMore")
			if state.moreBusy {
				text = a.t("browser.loadingMore")
			}
			button := ui.Button(c, text).FontSize(10.5).Disabled(state.moreBusy)
			if button.Clicked() {
				a.loadMore()
			}
		}
	})
}

// watchTransfers follows the transfer engine and reloads listings an
// upload changed.
func (a *App) watchTransfers() {
	a.svc.Transfers.OnUpdate = func(task core.TransferTask) {
		a.releaseStaged(task)
		a.update(func() {
			a.applyTask(task)
		})
	}
}

// applyTask merges a live transfer update into the page's state.
func (a *App) applyTask(task core.TransferTask) {
	index := -1
	for i := range a.transfers.tasks {
		if a.transfers.tasks[i].ID == task.ID {
			index = i
			break
		}
	}
	previous := ""
	if index >= 0 {
		previous = a.transfers.tasks[index].Status
		a.transfers.tasks[index] = task
	} else {
		a.transfers.tasks = append([]core.TransferTask{task}, a.transfers.tasks...)
	}
	if task.Direction == "upload" && task.Status == "completed" && previous != "completed" {
		if a.location != nil && a.location.ProfileID == task.Object.ProfileID && a.location.Bucket == task.Object.Bucket {
			a.loadEntries()
		}
	}
}

// loadTasks reads the transfer queue.
func (a *App) loadTasks() {
	go func() {
		tasks, err := a.svc.Tasks(a.ctx)
		if err != nil {
			return
		}
		a.update(func() {
			a.transfers.tasks = tasks
			a.transfers.loaded = true
		})
	}()
}

// pickAndUpload asks for files or a folder and uploads them.
func (a *App) pickAndUpload(directory bool) {
	location := a.location
	if location == nil {
		return
	}
	if a.readOnly() {
		a.notify(a.t("browser.readOnlyTip"))
		return
	}
	go func() {
		options := mygo.OpenDialogOptions{
			Parent: a.win, Title: a.t("browser.upload"),
			Multiple: !directory, Directory: directory,
		}
		if directory {
			options.Title = a.t("browser.uploadFolder")
		} else {
			options.Title = a.t("browser.uploadFiles")
		}
		paths, err := mygo.Dialog.Open(options)
		if err != nil || len(paths) == 0 {
			return
		}
		a.startUploads(*location, paths)
	}()
}

// startUploads queues the uploads of local paths and reports what happened.
func (a *App) startUploads(location Location, paths []string) {
	ctx := context.Background()
	go func() {
		results, err := a.uploadPaths(ctx, location, paths)
		a.update(func() {
			if err != nil && len(results) == 0 {
				a.toastCurrent(a.errorText(err))
				return
			}
			tasks := 0
			plans := []core.OperationPlan{}
			for _, result := range results {
				if result.Task != nil {
					tasks++
				}
				if result.Plan != nil {
					plans = append(plans, *result.Plan)
				}
			}
			if tasks > 0 {
				a.toastCurrent(a.tn("toast.queued", tasks, "count", tasks))
			}
			if len(plans) > 0 {
				a.toastCurrent(a.tn("toast.reviewPlans", len(plans), "count", len(plans)))
				for _, plan := range plans {
					a.queuePlan(planRequest{operation: "upload", location: location, plan: &plan})
				}
			}
			a.loadTasks()
		})
	}()
}

// notify shows a message without a view to attach it to.
func (a *App) notify(message string) {
	a.update(func() {
		a.pendingToast = message
	})
}
