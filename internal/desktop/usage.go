package desktop

import (
	"context"
	"sort"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// ensureUsage prepares the usage page for the folder in view.
func (a *App) ensureUsage() {
	if a.location == nil {
		return
	}
	if a.usage.location == nil || !sameLocation(a.usage.location, a.location) {
		a.usage.usage = core.Usage{}
		a.usage.progress = core.Usage{}
		a.usage.err = nil
		a.usage.location = a.location
	}
}

// usagePage draws the usage scan of the current folder or bucket.
func (a *App) usagePage(c *ui.Context) {
	a.ensureUsage()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).MaxWidth(880).Gap(16).Padding(20, 24).Margin(0, ui.Auto).Children(func() {
			ui.Text(c, a.t("usage.note")).FontSize(12).TextColor(a.pal.muted)
			a.usageControls(c)
			a.usageBody(c)
		})
	})
}

// usageControls start, stop and choose the scope of a scan.
func (a *App) usageControls(c *ui.Context) {
	if a.location == nil {
		return
	}
	location := *a.location
	scope := a.tf("usage.scopeBucket", "bucket", location.Bucket)
	if location.Prefix != "" {
		scope = a.t("usage.scopePrefix") + " · " + location.Prefix
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Badge(c, scope).Background(a.pal.field).TextColor(a.pal.muted)
		ui.Spacer(c)
		if a.usage.scanning {
			if ui.Button(c, a.t("usage.cancel")).Clicked() {
				a.cancelUsage()
			}
			return
		}
		label := a.t("usage.start")
		if !a.usage.usage.ScannedAt.IsZero() {
			label = a.t("usage.rescan")
		}
		if ui.PrimaryButton(c, label).Clicked() {
			a.startUsage(location)
		}
	})
}

// usageBody shows the totals and the breakdown of the last scan.
func (a *App) usageBody(c *ui.Context) {
	state := &a.usage
	switch {
	case state.scanning && state.progress.Objects == 0:
		ui.Column(c).Padding(48).Center().Gap(10).Children(func() {
			ui.Spinner(c).Label(a.t("usage.scanning"))
			ui.Text(c, a.t("usage.scanning")).FontSize(12.5).TextColor(a.pal.muted)
		})
		return
	case state.err != nil && state.usage.Objects == 0:
		a.errorCard(c, a.tf("usage.failed", "message", a.errorText(state.err)), func() {
			if a.location != nil {
				a.startUsage(*a.location)
			}
		})
		return
	case state.usage.ScannedAt.IsZero():
		a.emptyUsage(c)
		return
	}

	usage := state.usage
	if state.scanning {
		usage = state.progress
	}
	ui.Row(c).Gap(12).Children(func() {
		a.statCard(c, a.t("usage.objects"), formatCount(usage.Objects))
		a.statCard(c, a.t("usage.size"), formatBytes(usage.Bytes))
		a.statCard(c, a.t("usage.scannedAt"), formatTime(usage.ScannedAt))
		a.statCard(c, a.t("usage.completeness"), a.usageCompleteness())
	})
	if state.scanning {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(14, 14)
			ui.Text(c, a.tf("usage.progress", "count", formatCount(state.progress.Objects))).
				FontSize(12).TextColor(a.pal.muted)
		})
	} else if !usage.Complete {
		ui.Text(c, a.t("usage.partial")).FontSize(12).TextColor(a.pal.warningText)
	}
	if usage.Objects == 0 {
		ui.Text(c, a.t("usage.noObjects")).FontSize(12.5).TextColor(a.pal.muted)
		return
	}
	a.usageBreakdown(c, a.t("usage.byType"), usage.Types, true)
	a.usageBreakdown(c, a.t("usage.byExtension"), usage.Counts, false)
}

// usageCompleteness names whether the scan finished.
func (a *App) usageCompleteness() string {
	switch {
	case a.usage.scanning:
		return a.t("usage.scanning")
	case a.usage.usage.Complete:
		return a.t("usage.complete")
	}
	return a.t("usage.incomplete")
}

// statCard is one number of the scan.
func (a *App) statCard(c *ui.Context, label, value string) {
	card := ui.Column(c).Grow(1).Gap(4).Padding(12, 14).Radius(10).
		Background(a.pal.card).Border(1, a.pal.border)
	card.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(a.pal.faint)
		ui.Text(c, value).FontSize(16).Bold().SingleLine()
	})
}

// usageBreakdown shows one split of the scan as a bar and a list.
func (a *App) usageBreakdown(c *ui.Context, title string, counts map[string]int64, byKind bool) {
	type slice struct {
		name  string
		count int64
	}
	slices := make([]slice, 0, len(counts))
	var total int64
	for name, count := range counts {
		slices = append(slices, slice{name: name, count: count})
		total += count
	}
	sort.Slice(slices, func(i, j int) bool {
		if slices[i].count != slices[j].count {
			return slices[i].count > slices[j].count
		}
		return slices[i].name < slices[j].name
	})
	if len(slices) > 12 {
		slices = slices[:12]
	}

	card := ui.Column(c).Gap(10).Padding(16).Radius(12).
		Background(a.pal.card).Border(1, a.pal.border)
	card.Children(func() {
		ui.Text(c, title).FontSize(13).Bold()
		ui.Row(c).Height(10).Radius(5).Clip().Children(func() {
			for _, item := range slices {
				width := float32(0)
				if total > 0 {
					width = float32(item.count) / float32(total)
				}
				ui.Box(c).WidthPercent(width * 100).FillHeight().
					Background(a.usageColor(item.name, byKind))
			}
		})
		for _, item := range slices {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(10, 10).Radius(3).Background(a.usageColor(item.name, byKind))
				label := item.name
				if byKind {
					label = a.t("kinds." + item.name)
				}
				ui.Text(c, label).FontSize(12).SingleLine().Grow(1)
				ui.Text(c, formatCount(item.count)).FontSize(12).TextColor(a.pal.muted)
			})
		}
	})
}

// usageColor picks the color of one slice of the breakdown.
func (a *App) usageColor(name string, byKind bool) ui.Color {
	kind := name
	if !byKind {
		kind = kindOfExtension(name)
	}
	return a.pal.kindPaletteOf(kind).foreground
}

// emptyUsage invites the first scan.
func (a *App) emptyUsage(c *ui.Context) {
	if a.location == nil {
		ui.Column(c).Padding(48).Center().Children(func() {
			ui.Text(c, a.t("usage.noBucket")).FontSize(12.5).TextColor(a.pal.muted)
		})
		return
	}
	ui.Column(c).Padding(48).Center().Children(func() {
		ui.Column(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPieChart).FontSize(28).TextColor(a.pal.faint)
			ui.Text(c, a.t("usage.idleTitle")).FontSize(13.5).Bold()
			ui.Text(c, a.t("usage.idleDesc")).FontSize(12).TextColor(a.pal.muted).
				TextAlign(ui.Center)
		})
	})
}

// errorCard explains a failure with a way to try again.
func (a *App) errorCard(c *ui.Context, message string, retry func()) {
	ui.Column(c).Gap(8).Padding(20).Radius(12).AlignItems(ui.Center).
		Background(a.pal.destructiveBg).Children(func() {
		ui.Text(c, message).FontSize(12.5).TextColor(a.pal.destructive).TextAlign(ui.Center)
		if ui.Button(c, a.t("common.retry")).Clicked() {
			retry()
		}
	})
}

// startUsage scans the folder in view.
func (a *App) startUsage(location Location) {
	if a.usage.scanning {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.usage.scanning = true
	a.usage.cancel = cancel
	a.usage.err = nil
	a.usage.progress = core.Usage{}
	a.usage.location = &location
	go func() {
		usage, err := a.svc.ScanUsage(ctx, location.ProfileID, location.Bucket, location.Prefix,
			func(progress core.Usage) {
				a.update(func() {
					if !a.usage.scanning {
						return
					}
					a.usage.progress = progress
				})
			})
		a.update(func() {
			a.usage.scanning = false
			a.usage.cancel = nil
			if err != nil && !errCancelled(err) {
				a.usage.err = err
				a.toastCurrent(a.errorText(err))
				return
			}
			if errCancelled(err) {
				a.usage.usage = usage
				a.usage.usage.Complete = false
				return
			}
			a.usage.usage = usage
		})
	}()
}

// cancelUsage stops a scan, keeping what it found so far.
func (a *App) cancelUsage() {
	if a.usage.cancel != nil {
		a.usage.cancel()
	}
}
