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

// planOverlay draws the plan review over the window.
func (a *App) planOverlay(c *ui.Context) {
	state := &a.plan
	if !state.open {
		return
	}
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Background(a.pal.scrim)
		backdrop.Children(func() {
			panel := ui.Column(c).Width(640).MaxHeightPercent(88).Radius(16).
				Background(a.pal.background).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.22))
			panel.Clip()
			panel.Children(func() {
				a.planHeader(c)
				ui.Column(c).Grow(1).Children(func() { a.planBody(c) })
				a.planFooter(c)
			})
			if backdrop.OverlayShortcut(0, ui.KeyEscape) {
				a.closePlan()
			}
		})
	})
	if state.pickerOpen {
		a.pickerOverlay(c)
	}
}

// planHeader names the operation and where it applies.
func (a *App) planHeader(c *ui.Context) {
	state := &a.plan
	title := a.planTitle()
	ui.Row(c).Height(52).PaddingX(18).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		ui.Icon(c, planGlyph(state.operation)).FontSize(18).TextColor(a.planTone())
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(14).Bold().SingleLine()
			ui.Text(c, a.planSubtitle()).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
		})
		if len(state.queue) > 0 {
			ui.Badge(c, a.tn("plan.nextPlan", len(state.queue), "count", len(state.queue))).
				Background(a.pal.field).TextColor(a.pal.muted)
		}
		if a.iconAction(c, iconX, a.t("common.close")) {
			a.closePlan()
		}
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
		return iconArrowUpRight
	}
	return iconUpload
}

// planTone colors the plan's icon.
func (a *App) planTone() ui.Color {
	if a.plan.operation == "delete" {
		return a.pal.destructive
	}
	return a.pal.primary
}

// planBody shows what the plan will do.
func (a *App) planBody(c *ui.Context) {
	state := &a.plan
	switch {
	case state.creating:
		ui.Column(c).Height(240).Center().Gap(10).Children(func() {
			ui.Spinner(c).Label(a.t("plan.creating"))
			ui.Text(c, a.t("plan.creating")).FontSize(12.5).TextColor(a.pal.muted)
		})
	case state.err != nil:
		ui.Column(c).Height(240).Center().Padding(24).Children(func() {
			ui.Column(c).MaxWidth(380).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconCircleAlert).FontSize(28).TextColor(a.pal.destructive)
				ui.Text(c, a.errorText(state.err)).FontSize(12.5).
					TextColor(a.pal.muted).TextAlign(ui.Center)
			})
		})
	case state.running:
		a.planRunning(c)
	case !state.ready:
		return
	case state.expired():
		a.planExpired(c)
	default:
		a.planReview(c)
	}
}

// expired reports whether the plan's validity has run out.
func (s *planState) expired() bool {
	return s.ready && !s.plan.ExpiresAt.IsZero() && time.Now().After(s.plan.ExpiresAt)
}

// planReview lists the plan's totals, warnings and items.
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

	ui.Scroll(c).MaxHeight(420).Children(func() {
		ui.Column(c).Gap(12).Padding(18).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				a.planStat(c, a.t("plan.objects"), formatCount(int64(len(plan.Items))))
				a.planStat(c, a.t("plan.size"), formatBytes(plan.Bytes))
				if overwrite > 0 {
					a.planStat(c, a.t("plan.overwrite"), formatCount(int64(overwrite)))
				}
			})
			if plan.Warning != "" {
				a.planNotice(c, a.pal.warningBg, a.pal.warningText, plan.Warning)
			}
			switch state.operation {
			case "delete":
				a.planNotice(c, a.pal.destructiveBg, a.pal.destructive, a.t("plan.warnDelete"))
			case "move":
				a.planNotice(c, a.pal.warningBg, a.pal.warningText, a.t("plan.warnMove"))
			case "upload":
				if overwrite > 0 {
					a.planNotice(c, a.pal.warningBg, a.pal.warningText,
						a.tf("plan.warnOverwrite", "count", overwrite))
				}
			}
			if folders {
				a.planNotice(c, a.pal.field, a.pal.muted, a.t("plan.warnFolders"))
			}
			ui.Column(c).Gap(1).Children(func() {
				for i := range plan.Items {
					a.planItem(c, &plan.Items[i])
				}
			})
			ui.Text(c, a.t("plan.idNote")).FontSize(11).TextColor(a.pal.faint)
		})
	})
}

// planStat is one number of the plan's summary.
func (a *App) planStat(c *ui.Context, label, value string) {
	ui.Column(c).Grow(1).Gap(2).Padding(10, 12).Radius(10).
		Background(a.pal.field).Children(func() {
		ui.Text(c, label).FontSize(10.5).TextColor(a.pal.faint)
		ui.Text(c, value).FontSize(14).Bold().SingleLine()
	})
}

// planNotice is a warning line inside the plan.
func (a *App) planNotice(c *ui.Context, background, color ui.Color, text string) {
	ui.Row(c).Gap(8).Padding(8, 10).Radius(8).Background(background).
		AlignItems(ui.Start).Children(func() {
		ui.Icon(c, iconAlertTriangle).FontSize(14).TextColor(color)
		ui.Text(c, text).FontSize(11.5).TextColor(color).Grow(1)
	})
}

// planItem is one row of the plan.
func (a *App) planItem(c *ui.Context, item *core.PlanItem) {
	key := item.Source.Key
	if item.LocalPath != "" {
		key = item.LocalPath
	}
	ui.Row(c).Padding(6, 8).Gap(8).AlignItems(ui.Center).Radius(6).Children(func() {
		ui.Icon(c, iconOf(kindOf(item.Source.Key, item.Source.ContentType, false))).
			FontSize(14).TextColor(a.pal.muted)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, basename(key)).FontSize(12).SingleLine()
			if item.Target != nil && item.Target.Key != "" {
				ui.Text(c, item.Target.Key).FontSize(10.5).TextColor(a.pal.faint).SingleLine()
			}
		})
		if item.Existing != nil {
			ui.Badge(c, a.t("plan.overwrite")).Background(a.pal.warningBg).
				TextColor(a.pal.warningText)
		}
		ui.Text(c, formatBytes(item.Source.Size)).FontSize(11.5).TextColor(a.pal.muted)
	})
}

// planExpired explains that a plan has to be made again.
func (a *App) planExpired(c *ui.Context) {
	ui.Column(c).Height(220).Center().Gap(10).Children(func() {
		ui.Icon(c, iconClock).FontSize(28).TextColor(a.pal.warningText)
		ui.Text(c, a.t("plan.expired")).FontSize(12.5).TextColor(a.pal.muted)
		if ui.PrimaryButton(c, a.t("plan.regenerate")).Clicked() {
			a.createPlan()
		}
	})
}

// planRunning shows how far the plan has got.
func (a *App) planRunning(c *ui.Context) {
	state := &a.plan
	ratio := 0.0
	if state.total > 0 {
		ratio = float64(state.progress) / float64(state.total)
	}
	ui.Column(c).Padding(24).Gap(10).Children(func() {
		ui.Text(c, a.tf("plan.running", "done", state.progress, "total", state.total)).
			FontSize(12.5).TextColor(a.pal.muted)
		a.progressBar(c, ratio, 6, a.pal.primary)
	})
}

// planFooter carries the confirmation and the actions.
func (a *App) planFooter(c *ui.Context) {
	state := &a.plan
	if !state.ready || state.running || state.expired() || state.creating {
		if state.err != nil {
			ui.Row(c).Padding(12, 18).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("common.close")).Clicked() {
					a.closePlan()
				}
			})
		}
		return
	}
	ui.Column(c).Gap(10).Padding(14, 18).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.planExpiry()).FontSize(11).TextColor(a.pal.faint)
			ui.Spacer(c)
			if ui.Button(c, a.t("common.cancel")).Clicked() {
				a.closePlan()
			}
		})
		ui.Checkbox(c, &state.confirmed, a.t("plan.confirm"))
		label := a.t("plan.execute")
		if state.operation == "delete" {
			label = a.tn("plan.executeDelete", len(state.plan.Items), "count", len(state.plan.Items))
		}
		button := ui.PrimaryButton(c, label).FillWidth().Disabled(!state.confirmed)
		if state.operation == "delete" {
			button.Background(a.pal.destructive)
		}
		if button.Clicked() && state.confirmed {
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
