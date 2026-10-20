package desktop

import (
	"context"
	"sort"
	"strconv"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// ensureUsage prepares the usage page for the folder in view.
func (a *App) ensureUsage() {
	scope := a.usageScope()
	if scope == nil {
		return
	}
	if a.usage.location == nil || !sameLocation(a.usage.location, scope) {
		a.usage.usage = core.Usage{}
		a.usage.progress = core.Usage{}
		a.usage.err = nil
		a.usage.location = scope
	}
}

// usageScope is what a scan covers: the folder in view, or its whole bucket.
func (a *App) usageScope() *Location {
	if a.location == nil {
		return nil
	}
	scope := *a.location
	if a.usage.wholeBucket {
		scope.Prefix = ""
	}
	return &scope
}

// usageToolbar is the rest of the page's title row: what the scan covers,
// the choice of scope and the scan button.
func (a *App) usageToolbar(c *ui.Context) {
	scope := a.usageScope()
	if scope == nil {
		ui.Spacer(c)
		return
	}
	ui.Text(c, scope.Bucket+"/"+scope.Prefix).FontSize(12).Font("monospace").
		TextColor(a.pal.muted).SingleLine().Grow(1).MinWidth(0)
	ui.Row(c).Gap(10).AlignItems(ui.Center).Shrink(0).Children(func() {
		if a.location.Prefix != "" {
			index := 1
			if a.usage.wholeBucket {
				index = 0
			}
			labels := []string{
				a.tf("usage.scopeBucket", "bucket", a.location.Bucket), a.t("usage.scopePrefix"),
			}
			if a.segments(c, &index, a.t("usage.title"), labels) && !a.usage.scanning {
				a.usage.wholeBucket = index == 0
				a.ensureUsage()
			}
		}
		if a.usage.scanning {
			if a.secondaryButton(c, iconX, a.t("usage.cancel")).Clicked() {
				a.cancelUsage()
			}
			return
		}
		label := a.t("usage.start")
		if !a.usage.usage.ScannedAt.IsZero() {
			label = a.t("usage.rescan")
		}
		if a.primaryAction(c, iconPieChart, label, false).Clicked() {
			a.startUsage(*scope)
		}
	})
}

// usagePage draws the usage scan of the current folder or bucket.
func (a *App) usagePage(c *ui.Context) {
	a.ensureUsage()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).MaxWidth(920).Gap(18).Padding(22, 28, 48, 28).Children(func() {
			ui.Text(c, a.t("usage.note")).FontSize(12.5).TextColor(a.pal.muted)
			a.usageBody(c)
		})
	})
}

// usageBody shows the totals and the breakdown of the last scan.
func (a *App) usageBody(c *ui.Context) {
	state := &a.usage
	switch {
	case a.location == nil:
		ui.Text(c, a.t("usage.noBucket")).FontSize(12.5).TextColor(a.pal.muted)
		return
	case state.err != nil && state.usage.Objects == 0 && !state.scanning:
		a.errorCard(c, a.tf("usage.failed", "message", a.errorText(state.err)), func() {
			if scope := a.usageScope(); scope != nil {
				a.startUsage(*scope)
			}
		})
		return
	case state.usage.ScannedAt.IsZero() && !state.scanning:
		a.emptyUsage(c)
		return
	}

	usage := state.usage
	if state.scanning {
		usage = state.progress
	}
	a.usageStats(c, usage)
	if state.scanning {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(12, 12)
			ui.Text(c, a.tf("usage.progress", "count", formatCount(state.progress.Objects))).
				FontSize(12).TextColor(a.pal.muted).FontFeatures("tnum")
		})
	} else if !usage.Complete {
		ui.Row(c).Gap(8).Padding(9, 12).Radius(9).AlignItems(ui.Center).
			Background(a.pal.warningBg).Children(func() {
			ui.Icon(c, iconAlertTriangle).FontSize(13).TextColor(a.pal.warningText)
			ui.Text(c, a.t("usage.partial")).FontSize(12.5).TextColor(a.pal.warningText)
		})
	}
	if usage.Objects == 0 {
		if !state.scanning {
			ui.Text(c, a.t("usage.noObjects")).FontSize(12.5).TextColor(a.pal.muted)
		}
		return
	}
	ui.Row(c).Gap(28).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1.2).MinWidth(0).Gap(10).Children(func() {
			ui.Text(c, a.t("usage.byType")).FontSize(12.5).FontWeight(600)
			a.typeBreakdown(c, usage.Types)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
			ui.Text(c, a.t("usage.byExtension")).FontSize(12.5).FontWeight(600)
			a.extensionBreakdown(c, usage.Counts)
		})
	})
}

// usageStats is the row of totals, in one card divided between them.
func (a *App) usageStats(c *ui.Context, usage core.Usage) {
	completeness, tone := a.t("usage.complete"), a.pal.successText
	switch {
	case a.usage.scanning:
		completeness, tone = a.t("usage.scanning"), a.pal.text
	case !usage.Complete:
		completeness, tone = a.t("usage.incomplete"), a.pal.warningText
	}
	scanned := "—"
	if !usage.ScannedAt.IsZero() {
		scanned = formatTime(usage.ScannedAt)
	}
	stats := []struct {
		label, value string
		color        ui.Color
	}{
		{a.t("usage.objects"), formatCount(usage.Objects), a.pal.text},
		{a.t("usage.size"), formatBytes(usage.Bytes), a.pal.text},
		{a.t("usage.scannedAt"), scanned, a.pal.text},
		{a.t("usage.completeness"), completeness, tone},
	}
	ui.Row(c).Radius(12).Border(1, a.pal.border).Clip().Children(func() {
		for i, stat := range stats {
			cell := ui.Column(c).Grow(1).MinWidth(0).Gap(4).Padding(14, 16)
			if i < len(stats)-1 {
				cell.BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border)
			}
			cell.Children(func() {
				ui.Text(c, stat.label).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
				ui.Text(c, stat.value).FontSize(20).FontWeight(650).LetterSpacing(-0.2).
					FontFeatures("tnum").TextColor(stat.color).SingleLine()
			})
		}
	})
}

// countSlice is one share of a scan's object count.
type countSlice struct {
	name  string
	count int64
}

// sortedCounts orders the shares of a count, the largest first.
func sortedCounts(counts map[string]int64) ([]countSlice, int64) {
	slices := make([]countSlice, 0, len(counts))
	var total int64
	for name, count := range counts {
		slices = append(slices, countSlice{name: name, count: count})
		total += count
	}
	sort.Slice(slices, func(i, j int) bool {
		if slices[i].count != slices[j].count {
			return slices[i].count > slices[j].count
		}
		return slices[i].name < slices[j].name
	})
	return slices, total
}

// typeBreakdown splits the objects by kind, as one bar and a table.
func (a *App) typeBreakdown(c *ui.Context, counts map[string]int64) {
	slices, total := sortedCounts(counts)
	ui.Row(c).Height(10).Radius(5).Gap(2).Clip().Background(a.pal.mutedFill).Children(func() {
		for _, item := range slices {
			if total > 0 && item.count > 0 {
				ui.Box(c).Grow(float32(item.count)).FillHeight().Background(a.usageColor(item.name, true))
			}
		}
	})
	ui.Column(c).Children(func() {
		for _, item := range slices {
			share := float64(0)
			if total > 0 {
				share = float64(item.count) / float64(total) * 100
			}
			ui.Row(c).Gap(10).Padding(8, 0).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
				ui.Box(c).Size(9, 9).Radius(3).Background(a.usageColor(item.name, true))
				ui.Text(c, a.t("kinds."+item.name)).FontSize(12.5).SingleLine().Grow(1).MinWidth(0)
				ui.Text(c, formatCount(item.count)).FontSize(12.5).FontFeatures("tnum").
					TextAlign(ui.End).Width(84).SingleLine()
				ui.Text(c, strconv.FormatFloat(share, 'f', 1, 64)+"%").FontSize(12.5).
					FontFeatures("tnum").TextColor(a.pal.muted).TextAlign(ui.End).Width(52).SingleLine()
			})
		}
	})
}

// extensionBreakdown lists the most common extensions with bars scaled to
// the first.
func (a *App) extensionBreakdown(c *ui.Context, counts map[string]int64) {
	slices, _ := sortedCounts(counts)
	if len(slices) > 8 {
		slices = slices[:8]
	}
	ui.Column(c).Children(func() {
		for _, item := range slices {
			width := float32(0)
			if slices[0].count > 0 {
				width = float32(item.count) / float32(slices[0].count) * 100
			}
			ui.Row(c).Gap(10).Padding(8, 0).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
				ui.Text(c, item.name).FontSize(12).Font("monospace").Width(60).SingleLine()
				ui.Box(c).Grow(1).Height(6).Radius(3).Clip().Background(a.pal.mutedFill).Children(func() {
					ui.Box(c).WidthPercent(width).FillHeight().Radius(3).Background(a.pal.primary)
				})
				ui.Text(c, formatCount(item.count)).FontSize(12.5).FontFeatures("tnum").
					TextAlign(ui.End).Width(80).SingleLine()
			})
		}
	})
}

// usageColor picks the color of one kind in the breakdown.
func (a *App) usageColor(name string, byKind bool) ui.Color {
	kind := name
	if !byKind {
		kind = kindOfExtension(name)
	}
	return a.pal.kindPaletteOf(kind).foreground
}

// emptyUsage invites the first scan.
func (a *App) emptyUsage(c *ui.Context) {
	ui.Row(c).Gap(18).Padding(40, 28).Radius(12).AlignItems(ui.Center).
		Border(1, a.pal.border).Children(func() {
		ui.Box(c).Size(56, 56).Radius(16).Center().Shrink(0).Background(a.pal.primarySoft).Children(func() {
			ui.Icon(c, iconPieChart).FontSize(28).TextColor(a.pal.primaryText)
		})
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, a.t("usage.idleTitle")).FontSize(15).FontWeight(600)
			ui.Text(c, a.t("usage.idleDesc")).FontSize(12.5).TextColor(a.pal.muted)
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
