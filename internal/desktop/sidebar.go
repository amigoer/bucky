package desktop

import (
	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// sidebar draws the connections, the transfer tray and the page entries.
func (a *App) sidebar(c *ui.Context, top float32, bar ui.TitleBar) {
	width := a.sidebarWidthClamped()
	ui.Column(c).Width(width).FillHeight().PaddingX(10).Children(func() {
		a.sidebarHeader(c, top, bar)
		a.connections(c)
		ui.Spacer(c)
		a.sidebarFooter(c)
	})
}

// sidebarHeader carries the brand on the content header's line, so the
// navigation below it starts level with the content body.
func (a *App) sidebarHeader(c *ui.Context, top float32, bar ui.TitleBar) {
	ui.Column(c).Height(top+panelInset).Padding(panelInset, 0, 0, 0).Children(func() {
		ui.Row(c).Height(top).PaddingStart(bar.Left + 4).AlignItems(ui.Center).
			DragWindow().Children(func() {
			a.brand(c)
			ui.Spacer(c)
		})
	})
}

// connections lists the saved connections and their buckets.
func (a *App) connections(c *ui.Context) {
	ui.Column(c).Grow(1).Gap(6).Padding(0, 0, 8, 0).Children(func() {
		ui.Row(c).PaddingX(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.t("sidebar.connections")).FontSize(11).Bold().
				TextColor(a.pal.faint).Grow(1)
			a.ghostButton(c, iconPlus, a.t("sidebar.addConnection"), 22, false, func() {
				a.openConnect("")
			})
		})
		ui.Scroll(c).Grow(1).Gap(2).Children(func() {
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
	})
}

// connectionGroup shows one connection and, when open, its buckets.
func (a *App) connectionGroup(c *ui.Context, profile *core.Profile) {
	open := !a.prefs.Collapsed[profile.ID]
	state := a.buckets[profile.ID]
	bad := state != nil && state.err != nil && isCredentialError(state.err)

	row := ui.Row(c).Height(30).PaddingX(8).Gap(8).Radius(6).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(a.pal.hover)
	}
	row.Children(func() {
		a.connectionTile(c, 20)
		ui.Text(c, profile.Name).FontSize(13).Bold().SingleLine().Grow(1)
		if profile.ReadOnly {
			badge := ui.Badge(c, "").Background(a.pal.warningBg).TextColor(a.pal.warningText)
			badge.Children(func() { ui.Icon(c, iconLock).FontSize(11) })
			badge.Tooltip(a.t("sidebar.readOnly"))
		}
		if bad {
			ui.Badge(c, a.t("sidebar.attention")).
				Background(a.pal.destructiveBg).TextColor(a.pal.destructive)
		} else if !open && state != nil && state.loaded {
			ui.Text(c, formatCount(int64(len(state.buckets)))).FontSize(11.5).
				TextColor(a.pal.faint)
		}
		chevron := ui.Icon(c, iconChevronRight).FontSize(12).TextColor(a.pal.faint)
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
}

// bucketList shows the buckets of a connection, indented under it.
func (a *App) bucketList(c *ui.Context, profile *core.Profile, state *bucketState) {
	ui.Column(c).Margin(0, 0, 4, 26).PaddingStart(8).
		BorderWidth(0, 0, 0, 1).BorderColor(a.pal.borderStrong).Gap(1).Children(func() {
		switch {
		case state == nil || (state.busy && !state.loaded):
			ui.Row(c).Height(28).PaddingX(8).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(12, 12)
			})
		case state.err != nil:
			button := ui.Button(c, a.t("sidebar.bucketsFailed")).FontSize(11.5)
			button.Tooltip(a.errorText(state.err))
			if button.Clicked() {
				if isCredentialError(state.err) {
					a.openConnect(profile.ID)
				} else {
					a.loadBuckets(profile.ID)
				}
			}
		case len(state.buckets) == 0:
			ui.Row(c).Height(28).PaddingX(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("sidebar.noBuckets")).FontSize(11.5).TextColor(a.pal.faint)
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
	row := ui.Row(c).Height(28).PaddingX(8).Gap(8).Radius(6).AlignItems(ui.Center)
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
		text := ui.Text(c, bucket.Name).FontSize(12.5).SingleLine().Grow(1)
		if active {
			text.Bold().TextColor(a.pal.primaryText)
		}
	})
	if row.Clicked() {
		a.navigate(Location{ProfileID: profile.ID, Bucket: bucket.Name, Prefix: ""})
	}
}

// connectionTile is the provider mark shown beside a connection.
func (a *App) connectionTile(c *ui.Context, size float32) {
	ui.Box(c).Size(size, size).Radius(6).Center().
		Background(ui.Oklch(0.68, 0.17, 50)).Children(func() {
		ui.Text(c, "R2").FontSize(size * 0.42).Bold().TextColor(ui.Hex("#ffffff"))
	})
}

// sidebarFooter holds the transfer tray and the page entries.
func (a *App) sidebarFooter(c *ui.Context) {
	ui.Column(c).Gap(8).Padding(8, 0, 10, 0).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		a.transferTray(c)
		ui.Row(c).Gap(6).Children(func() {
			quick := ui.Button(c, "").Label(a.t("sidebar.quickUpload")).Grow(1).Height(32)
			quick.Background(a.pal.primarySoft)
			quick.Children(func() {
				ui.Icon(c, iconZap).FontSize(15).TextColor(a.pal.primaryText)
				ui.Text(c, a.t("sidebar.quickUpload")).FontSize(12.5).Bold().
					TextColor(a.pal.primaryText).SingleLine().Grow(1)
			})
			if quick.Clicked() {
				a.openQuick()
			}
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

// footerIcon is one of the two page buttons beside the image host.
func (a *App) footerIcon(c *ui.Context, glyph *ui.SVG, label string, active bool, action func()) {
	a.ghostButton(c, glyph, label, 32, active, action)
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

	card := ui.Column(c).Gap(6).Padding(8, 10).Radius(12).
		Background(a.pal.background).Border(1, a.pal.border)
	if a.page == pageTransfers {
		card.Border(1.5, a.pal.primary)
	}
	card.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(28, 28).Center().Children(func() {
				ui.Icon(c, glyph).FontSize(15).TextColor(tone)
			})
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, title).FontSize(12.5).Bold().SingleLine()
				ui.Text(c, sub).FontSize(11).TextColor(a.pal.muted).SingleLine()
			})
			ui.Icon(c, iconChevronRight).FontSize(12).TextColor(a.pal.faint)
		})
		a.progressBar(c, ratio, 3, ring)
	})
	if card.Clicked() {
		a.page = pageTransfers
		a.loadTasks()
	}
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
