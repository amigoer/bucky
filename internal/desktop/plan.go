package desktop

import (
	"context"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// openPlan starts planning an operation over the chosen rows.
func (a *App) openPlan(operation string, choice []entry) {
	if a.location == nil || len(choice) == 0 {
		return
	}
	if operation == "copy" || operation == "move" {
		a.openPicker(operation, choice)
		return
	}
	a.queuePlan(planRequest{operation: operation, location: *a.location, entries: choice})
}

// queuePlan adds a plan to the review queue, showing the first one.
func (a *App) queuePlan(request planRequest) {
	if request.plan != nil {
		a.showPlan(request)
		return
	}
	if a.plan.open {
		a.plan.queue = append(a.plan.queue, request)
		return
	}
	a.showPlan(request)
}

// showPlan opens the review for one request.
func (a *App) showPlan(request planRequest) {
	a.plan = planState{
		open:      true,
		operation: request.operation,
		location:  request.location,
		entries:   request.entries,
		queue:     a.plan.queue,
	}
	if request.plan != nil {
		a.plan.plan = *request.plan
		a.plan.ready = true
		return
	}
	a.createPlan()
}

// createPlan expands the choice and asks the service for a plan.
func (a *App) createPlan() {
	state := &a.plan
	state.creating = true
	state.err = nil
	operation := state.operation
	location := state.location
	choice := append([]entry{}, state.entries...)
	target := state.target
	base := state.base
	go func() {
		sources, err := a.expandEntries(a.ctx, location, choice)
		if err != nil {
			a.update(func() {
				state.creating = false
				state.err = err
			})
			return
		}
		if len(sources) == 0 {
			a.update(func() {
				state.creating = false
				state.err = core.Invalid("There is nothing to plan for")
			})
			return
		}
		plan, err := a.svc.CreatePlan(a.ctx, core.PlanRequest{
			Operation: operation, Sources: sources, Target: target, Base: base,
		})
		a.update(func() {
			state.creating = false
			if err != nil {
				state.err = err
				return
			}
			state.plan = plan
			state.ready = true
			state.confirmed = false
		})
	}()
}

// expandEntries turns the chosen rows into object references, reading every
// object below a folder.
func (a *App) expandEntries(ctx context.Context, location Location, choice []entry) ([]core.ObjectRef, error) {
	out := []core.ObjectRef{}
	seen := map[string]bool{}
	add := func(key string) error {
		if seen[key] {
			return nil
		}
		seen[key] = true
		out = append(out, core.ObjectRef{ProfileID: location.ProfileID, Bucket: location.Bucket, Key: key})
		if len(out) > maxPlanItems {
			return core.Invalid("A plan can contain up to 1,000 objects")
		}
		return nil
	}
	for _, row := range choice {
		if !row.folder {
			if err := add(row.key); err != nil {
				return nil, err
			}
			continue
		}
		cursor := ""
		for {
			result, err := a.svc.Find(ctx, core.FindRequest{
				ProfileID: location.ProfileID, Bucket: location.Bucket, Prefix: row.key,
				Limit: maxPlanItems + 1, Cursor: cursor,
			})
			if err != nil {
				return nil, err
			}
			for _, object := range result.Objects {
				if err := add(object.Key); err != nil {
					return nil, err
				}
			}
			if result.Complete || result.NextCursor == "" {
				break
			}
			cursor = result.NextCursor
		}
	}
	return out, nil
}

// planOverlay draws the plan review as a sheet from the title row.
func (a *App) planOverlay(c *ui.Context) {
	state := &a.plan
	if !state.open {
		return
	}
	a.dialog(c, 520, true, false, a.closePlan, func() {
		a.planHeader(c)
		ui.Column(c).Grow(1).MinHeight(0).Children(func() { a.planBody(c) })
		a.planFooter(c)
	})
	if state.pickerOpen {
		a.pickerOverlay(c)
	}
}

// planHeader names the operation, the plan and how long it stays valid.
func (a *App) planHeader(c *ui.Context) {
	state := &a.plan
	tile, ink := a.planTone()
	subtitle := a.planSubtitle()
	if state.ready {
		subtitle = state.plan.ID
	}
	a.dialogHeader(c, planGlyph(state.operation), tile, ink, a.planTitle(), subtitle, func() {
		if len(state.queue) > 0 {
			a.chip(c, nil, a.tn("plan.nextPlan", len(state.queue), "count", len(state.queue)),
				a.pal.mutedFill, a.pal.muted)
		}
		if state.ready && !state.done() && !state.running {
			background, color := a.pal.mutedFill, a.pal.muted
			if left := time.Until(state.plan.ExpiresAt); left <= 0 {
				background, color = a.pal.destructiveBg, a.pal.destructive
			} else if left < time.Minute {
				background, color = a.pal.warningBg, a.pal.warningText
			}
			a.chip(c, iconClock, a.planExpiry(), background, color)
		}
	}, a.closePlan)
}

// chip is a small rounded label, with an icon when glyph is set.
func (a *App) chip(c *ui.Context, glyph *ui.SVG, label string, background, color ui.Color) {
	ui.Row(c).Height(22).PaddingX(9).Gap(5).Radius(11).AlignItems(ui.Center).Shrink(0).
		Background(background).Children(func() {
		if glyph != nil {
			ui.Icon(c, glyph).FontSize(11).TextColor(color)
		}
		ui.Text(c, label).FontSize(11.5).FontWeight(600).FontFeatures("tnum").TextColor(color).SingleLine()
	})
}

// planTitle names what the plan will do.
func (a *App) planTitle() string {
	state := &a.plan
	count := len(state.entries)
	if state.ready {
		count = len(state.plan.Items)
	}
	switch state.operation {
	case "delete":
		return a.tn("plan.titleDelete", count, "count", count)
	case "move":
		return a.tn("plan.titleMove", count, "count", count)
	case "copy":
		return a.tn("plan.titleCopy", count, "count", count)
	default:
		return a.tn("plan.titleUpload", count, "count", count)
	}
}

// planSubtitle names the destination or the connection in view.
func (a *App) planSubtitle() string {
	state := &a.plan
	if state.target.Key != "" {
		return a.tf("plan.to", "place", state.target.Key)
	}
	if state.target.Bucket != "" {
		return a.tf("plan.to", "place", state.target.Bucket)
	}
	if state.location.Prefix != "" {
		return state.location.Bucket + "/" + state.location.Prefix
	}
	return state.location.Bucket
}

// planGlyph is the icon of an operation.
func planGlyph(operation string) *ui.SVG {
	switch operation {
	case "delete":
		return iconTrash
	case "move":
		return iconArrowRightLeft
	case "copy":
		return iconCopy
	}
	return iconUpload
}

// planTone is the tile and ink of the plan's icon: red for a delete, amber
// for an upload that overwrites.
func (a *App) planTone() (ui.Color, ui.Color) {
	switch a.plan.operation {
	case "delete":
		return a.pal.destructiveBg, a.pal.destructive
	case "upload":
		return a.pal.warningBg, a.pal.warningText
	}
	return a.pal.primarySoft, a.pal.primaryText
}

// planBody shows what the plan will do, or what it did.
func (a *App) planBody(c *ui.Context) {
	state := &a.plan
	switch {
	case state.creating:
		ui.Row(c).Gap(8).Padding(6, 20, 26, 20).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(12, 12).Label(a.t("plan.creating"))
			ui.Text(c, a.t("plan.creating")).FontSize(13).TextColor(a.pal.muted)
		})
	case state.err != nil:
		a.planError(c)
	case state.running:
		a.planRunning(c)
	case !state.ready:
		return
	case state.done():
		a.planDone(c)
	default:
		a.planReview(c)
	}
}

// expired reports whether the plan's validity has run out.
func (s *planState) expired() bool {
	return s.ready && !s.plan.ExpiresAt.IsZero() && time.Now().After(s.plan.ExpiresAt)
}

// done reports whether the plan has run.
func (s *planState) done() bool {
	return s.ready && (s.plan.Status == "completed" || s.plan.Status == "partial")
}

// planError explains why the plan cannot be made or run. A plan whose
// objects changed can be made again.
func (a *App) planError(c *ui.Context) {
	state := &a.plan
	ui.Column(c).Gap(10).Padding(0, 20, 18, 20).Children(func() {
		switch core.AsError(state.err).Code {
		case "conflict", "expired":
			message := a.t("plan.changed")
			if core.AsError(state.err).Code == "expired" {
				message = a.t("plan.expired")
			}
			a.notice(c, a.pal.warningBg, a.pal.warningText, message)
			ui.Row(c).Children(func() {
				if a.secondaryButton(c, iconRefresh, a.t("plan.regenerate")).Clicked() {
					state.err = nil
					a.createPlan()
				}
			})
		default:
			a.notice(c, a.pal.destructiveBg, a.pal.destructive, a.errorText(state.err))
		}
	})
}

// planReview lists the plan's facts, warnings and the objects it covers.
func (a *App) planReview(c *ui.Context) {
	state := &a.plan
	plan := state.plan
	overwrite := 0
	for _, item := range plan.Items {
		if item.Existing != nil {
			overwrite++
		}
	}
	folders := false
	for _, row := range state.entries {
		if row.folder {
			folders = true
		}
	}

	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(10).Padding(0, 20, 14, 20).Children(func() {
			a.planFacts(c, overwrite)
			if state.expired() {
				ui.Row(c).Gap(10).Padding(9, 12).Radius(9).AlignItems(ui.Center).
					Background(a.pal.destructiveBg).Children(func() {
					ui.Icon(c, iconAlertTriangle).FontSize(13).TextColor(a.pal.destructive)
					ui.Text(c, a.t("plan.expired")).FontSize(12.5).FontWeight(600).
						TextColor(a.pal.destructive).Grow(1)
					if a.secondaryButton(c, nil, a.t("plan.regenerate")).Height(26).Clicked() {
						a.createPlan()
					}
				})
			}
			switch state.operation {
			case "delete":
				a.notice(c, a.pal.destructiveBg, a.pal.destructive, a.t("plan.warnDelete"))
			case "move":
				a.notice(c, a.pal.warningBg, a.pal.warningText, a.t("plan.warnMove"))
			}
			if overwrite > 0 {
				a.notice(c, a.pal.warningBg, a.pal.warningText, a.tf("plan.warnOverwrite", "count", overwrite))
			}
			if folders {
				a.notice(c, a.pal.mutedFill, a.pal.muted, a.t("plan.warnFolders"))
			}
			if plan.Warning != "" {
				a.notice(c, a.pal.mutedFill, a.pal.muted, plan.Warning)
			}
			a.planNames(c, plan.Items)
			ui.Text(c, a.t("plan.idNote")).FontSize(11.5).LineHeight(1.5).TextColor(a.pal.faint)
		})
	})
}

// planFacts is the grid of what the plan covers and where.
func (a *App) planFacts(c *ui.Context, overwrite int) {
	state := &a.plan
	plan := state.plan
	source := state.location.Bucket + "/" + state.location.Prefix
	target := "—"
	switch {
	case state.operation == "upload":
		source, target = a.tf("plan.localFiles", "count", len(plan.Items)), source
	case state.target.Bucket != "":
		target = state.target.Bucket + "/" + state.target.Key
	case state.operation != "delete":
		target = a.t("plan.notSelected")
	}
	connection := ""
	if profile := a.profileOf(state.location.ProfileID); profile != nil {
		connection = profile.Name
	}
	overwriteColor := a.pal.text
	if overwrite > 0 {
		overwriteColor = a.pal.warningText
	}
	facts := []struct {
		label, value string
		color        ui.Color
	}{
		{a.t("plan.objects"), formatCount(int64(len(plan.Items))), a.pal.text},
		{a.t("plan.size"), formatBytes(plan.Bytes), a.pal.text},
		{a.t("plan.overwrite"), formatCount(int64(overwrite)), overwriteColor},
		{a.t("plan.source"), source, a.pal.text},
		{a.t("plan.destination"), target, a.pal.text},
		{a.t("plan.connection"), connection, a.pal.text},
	}
	ui.Column(c).Radius(10).Border(1, a.pal.border).Clip().Shrink(0).Children(func() {
		for row := 0; row < len(facts); row += 3 {
			line := ui.Row(c)
			if row == 0 {
				line.BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border)
			}
			line.Children(func() {
				for i := row; i < row+3; i++ {
					fact := facts[i]
					cell := ui.Column(c).Grow(1).MinWidth(0).Gap(2).Padding(9, 12)
					if i < row+2 {
						cell.BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border)
					}
					cell.Tooltip(fact.value).Children(func() {
						ui.Text(c, fact.label).FontSize(11).TextColor(a.pal.muted).SingleLine()
						ui.Text(c, fact.value).FontSize(13).FontWeight(600).TextColor(fact.color).SingleLine()
					})
				}
			})
		}
	})
}

// planNames shows the first objects of the plan by name.
func (a *App) planNames(c *ui.Context, items []core.PlanItem) {
	const shown = 6
	ui.Row(c).Gap(5).Wrap().Children(func() {
		for i, item := range items {
			if i == shown {
				a.nameChip(c, "+"+formatCount(int64(len(items)-shown)))
				break
			}
			key := item.Source.Key
			if item.LocalPath != "" {
				key = item.LocalPath
			}
			a.nameChip(c, basename(key))
		}
	})
}

func (a *App) nameChip(c *ui.Context, name string) {
	ui.Text(c, name).FontSize(11.5).Font("monospace").SingleLine().MaxWidth(200).
		Height(22).PaddingX(8).Radius(6).Background(a.pal.mutedFill)
}

// planRunning shows how far the plan has got.
func (a *App) planRunning(c *ui.Context) {
	state := &a.plan
	ratio := 0.0
	if state.total > 0 {
		ratio = float64(state.progress) / float64(state.total)
	}
	ui.Column(c).Padding(6, 20, 26, 20).Gap(10).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(12, 12)
			ui.Text(c, a.tf("plan.running", "done", state.progress, "total", state.total)).
				FontSize(13).FontFeatures("tnum")
		})
		a.progressBar(c, ratio, 5, a.pal.primary)
	})
}

// planDone shows what each item of the plan came to.
func (a *App) planDone(c *ui.Context) {
	results := a.plan.plan.Results
	succeeded, failed := 0, 0
	for _, result := range results {
		if result.Error != nil {
			failed++
		} else {
			succeeded++
		}
	}
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(10).Padding(0, 20, 14, 20).Children(func() {
			background, color, glyph, title := a.pal.successBg, a.pal.successText, iconCheck, a.t("plan.allDone")
			if failed > 0 {
				background, color, glyph = a.pal.warningBg, a.pal.warningText, iconAlertTriangle
				title = a.tf("plan.partial", "count", failed)
			}
			ui.Row(c).Gap(9).Padding(10, 12).Radius(10).AlignItems(ui.Center).
				Background(background).Children(func() {
				ui.Icon(c, glyph).FontSize(15).TextColor(color)
				ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(color)
			})
			ui.Row(c).Gap(8).Children(func() {
				a.resultCount(c, a.t("plan.ok"), succeeded, a.pal.successText)
				failColor := a.pal.text
				if failed > 0 {
					failColor = a.pal.destructive
				}
				a.resultCount(c, a.t("plan.fail"), failed, failColor)
			})
			ui.Column(c).Radius(10).Border(1, a.pal.border).Clip().Children(func() {
				for i := range results {
					a.resultRow(c, &results[i], i < len(results)-1)
				}
			})
		})
	})
}

func (a *App) resultCount(c *ui.Context, label string, count int, color ui.Color) {
	ui.Column(c).Grow(1).Gap(1).Padding(8, 12).Radius(9).Border(1, a.pal.border).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(a.pal.muted)
		ui.Text(c, formatCount(int64(count))).FontSize(18).FontWeight(650).FontFeatures("tnum").TextColor(color)
	})
}

// resultRow is what one item of an executed plan came to.
func (a *App) resultRow(c *ui.Context, result *core.ItemResult, divider bool) {
	key := result.Source.Key
	if key == "" && result.Target != nil {
		key = result.Target.Key
	}
	label, background, color := a.t("plan.ok"), a.pal.successBg, a.pal.successText
	switch {
	case result.Error != nil:
		label, background, color = a.t("plan.fail"), a.pal.destructiveBg, a.pal.destructive
	case result.Status != "completed":
		label, background, color = a.t("transfers.status.queued"), a.pal.mutedFill, a.pal.muted
	}
	row := ui.Row(c).Gap(10).Padding(8, 12).AlignItems(ui.Center)
	if divider {
		row.BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border)
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(10.5).FontWeight(600).TextColor(color).TextAlign(ui.Center).
			Width(52).Height(18).Radius(9).Background(background).SingleLine()
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, key).FontSize(11.5).Font("monospace").SingleLine()
			if result.Error != nil {
				ui.Text(c, a.errorText(result.Error)).FontSize(11.5).TextColor(a.pal.muted)
			}
		})
	})
}

// planFooter carries the confirmation and the actions.
func (a *App) planFooter(c *ui.Context) {
	state := &a.plan
	switch {
	case state.done(), state.err != nil:
		a.dialogFooter(c, func() {
			ui.Spacer(c)
			if a.secondaryButton(c, nil, a.t("common.done")).Height(30).Clicked() {
				a.closePlan()
			}
		})
		return
	case !state.ready || state.running || state.creating:
		return
	}
	usable := !state.expired()
	a.dialogFooter(c, func() {
		check := ui.Row(c).Grow(1).Gap(9).AlignItems(ui.Center).Label(a.t("plan.confirm"))
		if usable {
			check.Cursor(ui.CursorPointer)
		} else {
			check.Opacity(0.4)
		}
		check.Children(func() {
			box := ui.Box(c).Size(16, 16).Radius(5).Center().Shrink(0)
			if state.confirmed && usable {
				box.Background(a.pal.primary).Children(func() {
					ui.Icon(c, iconCheck).FontSize(11).TextColor(a.pal.onPrimary)
				})
			} else {
				box.Border(1.5, a.pal.borderStrong)
			}
			ui.Text(c, a.t("plan.confirm")).FontSize(12.5)
		})
		if usable && check.Clicked() {
			state.confirmed = !state.confirmed
		}
		if a.secondaryButton(c, nil, a.t("common.cancel")).Height(30).Clicked() {
			a.closePlan()
		}
		label := a.t("plan.execute")
		if state.operation == "delete" {
			label = a.tn("plan.executeDelete", len(state.plan.Items), "count", len(state.plan.Items))
		}
		ready := state.confirmed && usable
		button := a.primaryAction(c, nil, label, !ready).Height(30)
		if state.operation == "delete" {
			button.Background(a.pal.destructive)
		}
		if ready && button.Clicked() {
			a.executePlan()
		}
	})
}

// planExpiry writes how long the plan stays valid.
func (a *App) planExpiry() string {
	plan := a.plan.plan
	if plan.ExpiresAt.IsZero() {
		return ""
	}
	left := time.Until(plan.ExpiresAt)
	if left <= 0 {
		return a.t("plan.expiredChip")
	}
	return a.tf("plan.expiresIn", "time", formatDuration(left.Seconds()))
}

// executePlan runs a reviewed plan.
func (a *App) executePlan() {
	state := &a.plan
	state.running = true
	state.progress = 0
	state.total = len(state.plan.Items)
	state.err = nil
	id := state.plan.ID
	go func() {
		plan, err := a.svc.ExecutePlan(a.ctx, id, true)
		a.update(func() {
			state.running = false
			if err != nil {
				state.err = err
				a.toastCurrent(a.errorText(err))
				return
			}
			state.plan = plan
			state.ready = true
			state.confirmed = false
			failed := 0
			for _, result := range plan.Results {
				if result.Error != nil {
					failed++
				}
			}
			if failed == 0 {
				a.toastCurrent(a.t("toast.planDone"))
			} else {
				a.toastCurrent(a.t("toast.planPartial"))
			}
			a.loadTasks()
			if a.location != nil && a.location.ProfileID == state.location.ProfileID &&
				a.location.Bucket == state.location.Bucket {
				a.loadEntries()
			}
		})
	}()
}

// closePlan closes the review and opens the next one in the queue.
func (a *App) closePlan() {
	queue := a.plan.queue
	a.plan = planState{}
	if len(queue) == 0 {
		return
	}
	next := queue[0]
	a.plan.queue = queue[1:]
	a.showPlan(next)
}

// openPicker starts choosing a destination for a copy or a move.
func (a *App) openPicker(operation string, choice []entry) {
	if a.location == nil {
		return
	}
	location := *a.location
	a.plan = planState{
		open:      true,
		operation: operation,
		location:  location,
		entries:   choice,
		queue:     a.plan.queue,
	}
	a.plan.pickerOpen = true
	a.plan.picker = pickerState{location: location, path: breadcrumbs(location.Prefix)}
	a.loadPicker()
}

// loadPicker reads the folders of the destination being chosen.
func (a *App) loadPicker() {
	state := &a.plan.picker
	state.loading = true
	state.err = nil
	location := state.location
	go func() {
		page, err := a.svc.List(a.ctx, core.ListRequest{
			ProfileID: location.ProfileID, Bucket: location.Bucket,
			Prefix: location.Prefix, Delimiter: "/", Limit: pageSize,
		})
		a.update(func() {
			state.loading = false
			if err != nil {
				state.err = err
				return
			}
			rows := entriesFromPages([]core.ListPage{page}, location.Prefix)
			folders := make([]entry, 0, len(rows))
			for _, row := range rows {
				if row.folder {
					folders = append(folders, row)
				}
			}
			state.rows = sortEntries(folders, "name", 1)
		})
	}()
}

// pickerOverlay draws the destination chooser over the plan.
func (a *App) pickerOverlay(c *ui.Context) {
	state := &a.plan.picker
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Background(a.pal.scrim)
		backdrop.Children(func() {
			panel := ui.Column(c).Width(520).MaxHeightPercent(70).Radius(16).
				Background(a.pal.background).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.22))
			panel.Clip()
			panel.Children(func() {
				ui.Row(c).Height(48).PaddingX(16).Gap(10).AlignItems(ui.Center).
					BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
					ui.Text(c, a.t("plan.pick")).FontSize(13.5).Bold().Grow(1)
					if a.iconAction(c, iconX, a.t("common.close")) {
						a.closePlan()
					}
				})
				ui.Row(c).Padding(10, 14).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconDatabase).FontSize(14).TextColor(a.pal.faint)
					ui.Text(c, state.location.Bucket+"/"+state.location.Prefix).
						FontSize(11.5).TextColor(a.pal.muted).SingleLine().Grow(1)
				})
				ui.Column(c).Grow(1).Children(func() {
					switch {
					case state.loading:
						ui.Column(c).Height(180).Center().Children(func() { ui.Spinner(c) })
					case state.err != nil:
						ui.Column(c).Height(180).Center().Children(func() {
							ui.Text(c, a.errorText(state.err)).FontSize(12).
								TextColor(a.pal.muted)
						})
					default:
						ui.Scroll(c).MaxHeight(300).PaddingX(8).Children(func() {
							ui.Column(c).Gap(1).Children(func() {
								if state.location.Prefix != "" {
									if ui.Button(c, ".. "+a.t("common.back")).Clicked() {
										a.pickerEnter(parentPrefix(state.location.Prefix))
									}
								}
								for i := range state.rows {
									row := state.rows[i]
									item := ui.Row(c).Padding(6, 10).Gap(8).Radius(6).
										AlignItems(ui.Center)
									if item.Hovered() {
										item.Background(a.pal.hover)
									}
									item.Children(func() {
										ui.Icon(c, iconFolder).FontSize(15).
											TextColor(a.pal.kindPaletteOf("folder").foreground)
										ui.Text(c, row.name).FontSize(12.5).SingleLine().Grow(1)
									})
									if item.Clicked() {
										a.pickerEnter(row.key)
									}
								}
								if len(state.rows) == 0 {
									ui.Text(c, a.t("states.emptyFolderTitle")).FontSize(12).
										TextColor(a.pal.faint).Padding(12)
								}
							})
						})
					}
				})
				ui.Row(c).Padding(12, 16).Gap(8).Justify(ui.End).
					BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
					if ui.Button(c, a.t("common.cancel")).Clicked() {
						a.closePlan()
					}
					if ui.PrimaryButton(c, a.t("plan.create")).Clicked() {
						a.confirmPicker()
					}
				})
			})
		})
	})
}

// pickerEnter opens a folder inside the destination chooser.
func (a *App) pickerEnter(prefix string) {
	state := &a.plan.picker
	state.location.Prefix = prefix
	state.path = breadcrumbs(prefix)
	a.loadPicker()
}

// confirmPicker takes the chosen folder as the destination and plans the
// copy or the move.
func (a *App) confirmPicker() {
	state := &a.plan
	picker := state.picker
	if picker.location.Prefix == state.location.Prefix {
		state.err = core.Invalid("Choose a destination folder other than the current one")
		state.pickerOpen = false
		return
	}
	if state.operation == "move" || state.operation == "copy" {
		for _, row := range state.entries {
			if row.folder && strings.HasPrefix(picker.location.Prefix, row.key) {
				state.err = core.Invalid("A folder cannot be moved or copied into itself")
				state.pickerOpen = false
				return
			}
		}
	}
	state.target = core.ObjectRef{
		ProfileID: picker.location.ProfileID,
		Bucket:    picker.location.Bucket,
		Key:       picker.location.Prefix,
	}
	state.base = state.location.Prefix
	state.pickerOpen = false
	a.createPlan()
}
