package desktop

import (
	"strings"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// sidebar draws the connections, the transfer tray and the page entries.
func (a *App) sidebar(c *ui.Context, top float32, bar ui.TitleBar) {
	width := a.sidebarWidthClamped()
	ui.Column(c).Width(width).FillHeight().Background(a.pal.sidebar).
		BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border).Children(func() {
		a.sidebarHeader(c, top, bar)
		a.connections(c)
		a.sidebarFooter(c)
	})
}

// sidebarHeader carries the window controls, the brand and the toggle.
func (a *App) sidebarHeader(c *ui.Context, top float32, bar ui.TitleBar) {
	start := float32(18)
	if controls := a.windowControlsWidth(bar); controls > 0 {
		start = trafficLightLeft + controls + 12
	}
	ui.Row(c).Height(top).Padding(0, 10, 0, start).Gap(8).AlignItems(ui.Center).
		DragWindow().Children(func() {
		a.brand(c)
		ui.Spacer(c)
		a.sidebarToggle(c)
	})
}

// connections lists the saved connections and their buckets.
func (a *App) connections(c *ui.Context) {
	ui.Scroll(c).Grow(1).Padding(6, 10, 10, 10).Gap(8).Children(func() {
		ui.Row(c).Padding(2, 8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.t("sidebar.connections")).FontSize(11).FontWeight(600).
				TextColor(a.pal.faint).Grow(1)
			a.ghostButton(c, iconPlus, a.t("sidebar.addConnection"), 20, false, func() {
				a.openConnect("")
			})
		})
		if a.profilesBusy && len(a.profiles) == 0 {
			ui.Row(c).Padding(8).Children(func() { ui.Spinner(c) })
			return
		}
		if a.profilesErr != nil {
			ui.Text(c, a.errorText(a.profilesErr)).FontSize(11.5).
				TextColor(a.pal.muted).Padding(8)
			return
		}
		for i := range a.profiles {
			a.connectionGroup(c, &a.profiles[i])
		}
	})
}

// connectionGroup shows one connection and, when open, its buckets.
func (a *App) connectionGroup(c *ui.Context, profile *core.Profile) {
	open := !a.prefs.Collapsed[profile.ID]
	state := a.buckets[profile.ID]
	bad := state != nil && state.err != nil && isCredentialError(state.err)

	ui.Column(c).Children(func() {
		row := ui.Row(c).Height(30).Padding(0, 6, 0, 8).Gap(9).Radius(7).AlignItems(ui.Center)
		if row.Hovered() {
			row.Background(a.pal.hover)
		}
		row.Tooltip(a.t("sidebar.editHint"))
		row.Children(func() {
			a.providerTile(c, profile.ProviderType, 20)
			ui.Text(c, profile.Name).FontSize(13).FontWeight(600).SingleLine().Grow(1)
			switch {
			case profile.ReadOnly:
				a.sidebarBadge(c, iconLock, a.t("sidebar.readOnly"), a.pal.warningBg, a.pal.warningText)
			case bad:
				a.sidebarBadge(c, iconAlertTriangle, a.t("sidebar.attention"), a.pal.destructiveBg, a.pal.destructive)
			case !open && state != nil && state.loaded:
				ui.Text(c, formatCount(int64(len(state.buckets)))).FontSize(11.5).
					TextColor(a.pal.faint)
			}
			chevron := ui.Icon(c, iconChevronRight).FontSize(10).TextColor(a.pal.faint)
			if open {
				chevron.Rotate(90)
			}
		})
		if row.DoubleClicked() {
			a.openConnect(profile.ID)
			return
		}
		if row.Clicked() {
			a.prefs.Collapsed[profile.ID] = open
			a.savePrefs()
			if open && profile.Bucket == "" {
				a.ensureBuckets(profile.ID)
			}
		}
		if !open {
			return
		}
		a.ensureBuckets(profile.ID)
		a.bucketList(c, profile, state)
	})
}

// sidebarBadge is the small label beside a connection that needs a look.
func (a *App) sidebarBadge(c *ui.Context, glyph *ui.SVG, label string, background, color ui.Color) {
	ui.Row(c).Height(18).PaddingX(6).Gap(3).Radius(5).AlignItems(ui.Center).
		Background(background).Children(func() {
		ui.Icon(c, glyph).FontSize(10).TextColor(color)
		ui.Text(c, label).FontSize(10.5).FontWeight(600).TextColor(color).SingleLine()
	})
}

// bucketList shows the buckets of a connection, indented under it.
func (a *App) bucketList(c *ui.Context, profile *core.Profile, state *bucketState) {
	ui.Column(c).Margin(2, 0, 2, 17).PaddingStart(9).
		BorderWidth(0, 0, 0, 1).BorderColor(a.pal.borderStrong).Gap(1).Children(func() {
		switch {
		case state == nil || (state.busy && !state.loaded):
			ui.Row(c).Height(28).PaddingX(8).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(12, 12)
			})
		case state.err != nil:
			row := ui.Row(c).Height(28).PaddingX(8).Gap(8).Radius(7).AlignItems(ui.Center)
			if row.Hovered() {
				row.Background(a.pal.hover)
			}
			row.Tooltip(a.errorText(state.err))
			row.Children(func() {
				ui.Icon(c, iconAlertTriangle).FontSize(13).TextColor(a.pal.destructive)
				ui.Text(c, a.t("sidebar.bucketsFailed")).FontSize(12).TextColor(a.pal.muted).
					SingleLine().Grow(1)
			})
			if row.Clicked() {
				if isCredentialError(state.err) {
					a.openConnect(profile.ID)
				} else {
					a.loadBuckets(profile.ID)
				}
			}
		case len(state.buckets) == 0:
			ui.Row(c).Height(28).PaddingX(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("sidebar.noBuckets")).FontSize(12).TextColor(a.pal.faint)
			})
		default:
			for _, bucket := range state.buckets {
				a.bucketRow(c, profile, bucket)
			}
		}
	})
}

// bucketRow opens one bucket.
func (a *App) bucketRow(c *ui.Context, profile *core.Profile, bucket core.Bucket) {
	active := a.location != nil && a.location.ProfileID == profile.ID &&
		a.location.Bucket == bucket.Name && a.page == pageBrowse
	row := ui.Row(c).Height(28).PaddingX(8).Gap(8).Radius(7).AlignItems(ui.Center)
	if active {
		row.Background(a.pal.primarySoft)
	} else if row.Hovered() {
		row.Background(a.pal.hover)
	}
	row.Children(func() {
		color := a.pal.faint
		if active {
			color = a.pal.primary
		}
		ui.Icon(c, iconDatabase).FontSize(14).TextColor(color)
		text := ui.Text(c, bucket.Name).FontSize(13).SingleLine().Grow(1)
		if active {
			text.FontWeight(600).TextColor(a.pal.primaryText)
		}
	})
	if row.Clicked() {
		a.navigate(Location{ProfileID: profile.ID, Bucket: bucket.Name, Prefix: ""})
	}
}

// providerMarks are the letters and hue of each storage type's tile.
var providerMarks = map[string]struct {
	mark string
	hue  float32
}{
	"cos": {"COS", 250},
	"oss": {"OSS", 25},
	"s3":  {"S3", 85},
	"r2":  {"R2", 55},
}

// providerTile is the mark of a storage type, as the sidebar, the
// breadcrumbs and the connection form show it.
func (a *App) providerTile(c *ui.Context, providerType string, size float32) {
	mark, ok := providerMarks[providerType]
	background := ui.Oklch(0.56, 0.08, mark.hue)
	if !ok {
		mark.mark = strings.ToUpper(providerType)
		if len(mark.mark) > 3 {
			mark.mark = mark.mark[:3]
		}
		background = ui.Oklch(0.56, 0.02, 262)
	}
	ui.Box(c).Size(size, size).Radius(size * 0.3).Center().Background(background).Children(func() {
		ui.Text(c, mark.mark).FontSize(size * 0.4).FontWeight(700).
			LetterSpacing(-0.2).TextColor(ui.Hex("#ffffff")).SingleLine()
	})
}

// sidebarFooter holds the transfer tray and the page entries.
func (a *App) sidebarFooter(c *ui.Context) {
	ui.Column(c).Gap(8).Padding(8, 10, 10, 10).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		a.transferTray(c)
		ui.Row(c).Gap(6).Children(func() {
			a.quickUploadEntry(c)
			a.footerIcon(c, iconPieChart, a.t("sidebar.usage"), a.page == pageUsage, func() {
				a.page = pageUsage
				a.ensureUsage()
			})
			a.footerIcon(c, iconSliders, a.t("sidebar.settings"), a.page == pageSettings, func() {
				a.page = pageSettings
				a.loadSettings()
			})
		})
	})
}

// quickUploadEntry opens the image host, with its shortcut beside it.
func (a *App) quickUploadEntry(c *ui.Context) {
	label := a.t("sidebar.quickUpload")
	entry := ui.Row(c).Grow(1).Height(32).PaddingX(10).Gap(8).Radius(9).AlignItems(ui.Center).
		Label(label).Tooltip(label + " ⌥⌘U")
	if entry.Hovered() {
		entry.Background(a.pal.primaryLine)
	} else {
		entry.Background(a.pal.primarySoft)
	}
	entry.Children(func() {
		ui.Icon(c, iconZap).FontSize(15).TextColor(a.pal.primaryText)
		ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(a.pal.primaryText).
			SingleLine().Grow(1)
		if a.prefs.Language == langChinese {
			// The English label leaves no room for the shortcut, which its
			// tooltip names instead.
			ui.Text(c, "⌥⌘U").FontSize(10.5).TextColor(a.pal.primaryText).Opacity(0.75)
		}
	})
	if entry.Clicked() {
		a.openQuick()
	}
}

// footerIcon is one of the two page buttons beside the image host.
func (a *App) footerIcon(c *ui.Context, glyph *ui.SVG, label string, active bool, action func()) {
	button := a.plainButton(c, label, 32, 32).Radius(9)
	if active {
		button.Background(a.pal.hover)
	}
	color := a.pal.muted
	if active {
		color = a.pal.text
	}
	button.Children(func() { ui.Icon(c, glyph).FontSize(17).TextColor(color) })
	if button.Clicked() {
		action()
	}
}

// transferTray summarises the transfer queue and opens the transfers page.
func (a *App) transferTray(c *ui.Context) {
	moving, attention, speed, done, size := a.transferSummary()
	title := a.t("tray.idle")
	sub := a.t("tray.idleSub")
	tone := a.pal.successText
	ring := a.pal.success
	glyph := iconCheck
	ratio := 1.0
	switch {
	case moving > 0:
		title = a.tn("tray.active", moving, "count", moving)
		sub = a.tf("tray.activeSub", "speed", formatBytes(int64(speed)),
			"eta", formatDuration(etaSeconds(size-done, speed)))
		tone, ring, glyph = a.pal.primaryText, a.pal.primary, iconArrowDownUp
		if size > 0 {
			ratio = float64(done) / float64(size)
		}
	case attention > 0:
		title = a.tn("tray.attention", attention, "count", attention)
		sub = a.t("tray.attentionSub")
		tone, ring, glyph = a.pal.warningText, a.pal.warning, iconAlertTriangle
	}

	card := ui.Column(c).Gap(7).Padding(9, 10).Radius(11).Background(a.pal.background).
		Border(0.5, a.pal.borderStrong)
	if card.Hovered() {
		card.Shadow(0, 4, 12, 0, ui.RGBA(0, 0, 0, 0.08))
	} else {
		card.Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.05))
	}
	if a.page == pageTransfers {
		card.Border(1.5, a.pal.primary)
	}
	card.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			a.progressRing(c, ratio, ring, func() {
				ui.Icon(c, glyph).FontSize(12).TextColor(tone)
			})
			ui.Column(c).Grow(1).Gap(1).Children(func() {
				ui.Text(c, title).FontSize(12.5).FontWeight(600).SingleLine()
				ui.Text(c, sub).FontSize(11).TextColor(a.pal.muted).SingleLine()
			})
			ui.Icon(c, iconChevronRight).FontSize(10).TextColor(a.pal.faint)
		})
		a.progressBar(c, ratio, 3, ring)
	})
	if card.Clicked() {
		a.page = pageTransfers
		a.loadTasks()
	}
}

// progressRing is a 30 point ring filled clockwise to ratio, around a glyph.
func (a *App) progressRing(c *ui.Context, ratio float64, color ui.Color, glyph func()) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	at := float32(ratio)
	ui.Box(c).Size(30, 30).Radius(15).Center().ConicGradient(ui.ConicGradient{
		Stops: []ui.GradientStop{{Color: color, At: 0}, {Color: color, At: at},
			{Color: a.pal.mutedFill, At: at}, {Color: a.pal.mutedFill, At: 1}},
	}).Children(func() {
		ui.Box(c).Size(24, 24).Radius(12).Center().Background(a.pal.background).Children(glyph)
	})
}

// transferSummary counts the queue for the tray.
func (a *App) transferSummary() (moving, attention int, speed, done, size int64) {
	for _, task := range a.transfers.tasks {
		switch task.Status {
		case "queued", "running", "cancelling":
			moving++
			speed += task.BytesPerSecond
			size += task.Size
			if task.Completed < task.Size {
				done += task.Completed
			} else {
				done += task.Size
			}
		case "failed", "paused":
			attention++
		}
	}
	return moving, attention, speed, done, size
}

// etaSeconds is how long the rest of the queue takes at the current speed.
func etaSeconds(remaining, speed int64) float64 {
	if speed <= 0 || remaining <= 0 {
		return -1
	}
	return float64(remaining) / float64(speed)
}
