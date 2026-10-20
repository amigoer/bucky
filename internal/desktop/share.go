package desktop

import (
	"sort"

	"github.com/egoist/mygo/ui"
)

// shareDurations are the lifetimes the share dialog offers, in seconds.
var shareDurations = []int{900, 3600, 86400, 7 * 86400}

// openShare prepares a share for the chosen rows.
func (a *App) openShare(choice []entry) {
	if a.location == nil || len(choice) == 0 {
		return
	}
	mode := "temporary"
	if profile := a.currentProfile(); profile != nil && profile.PublicURL != "" {
		mode = "temporary"
	}
	a.share = shareState{
		open:     true,
		location: *a.location,
		entries:  append([]entry{}, choice...),
		mode:     mode,
		seconds:  3600,
	}
	a.buildShare()
}

// buildShare makes a link for every chosen object.
func (a *App) buildShare() {
	state := &a.share
	state.busy = true
	state.err = nil
	state.links = nil
	location := state.location
	choice := append([]entry{}, state.entries...)
	mode := state.mode
	seconds := state.seconds
	go func() {
		refs, err := a.expandEntries(a.ctx, location, choice)
		if err != nil {
			a.update(func() {
				state.busy = false
				state.err = err
			})
			return
		}
		links := make([]shareLink, 0, len(refs))
		for _, ref := range refs {
			link, err := a.svc.Share(a.ctx, ref, int64(seconds), mode == "public")
			item := shareLink{entry: entry{key: ref.Key, name: basename(ref.Key)}}
			if err != nil {
				item.err = err
			} else {
				item.url = link.URL
				item.headers = link.Headers
				if !link.ExpiresAt.IsZero() {
					item.expires = link.ExpiresAt.Local().Format("15:04")
				}
			}
			links = append(links, item)
		}
		a.update(func() {
			state.busy = false
			state.ready = true
			state.links = links
		})
	}()
}

// shareLinksShown is how many links the dialog lists; Copy all takes the
// rest too.
const shareLinksShown = 6

// shareOverlay draws the share dialog over the window.
func (a *App) shareOverlay(c *ui.Context) {
	state := &a.share
	if !state.open {
		return
	}
	closeShare := func() { a.share = shareState{} }
	a.dialog(c, 540, false, false, closeShare, func() {
		title := a.tn("share.objects", len(state.entries), "count", len(state.entries))
		if len(state.entries) == 1 {
			title = state.entries[0].name
			if state.entries[0].folder {
				title += "/"
			}
		}
		subtitle := state.location.Bucket + "/" + state.location.Prefix
		if profile := a.profileOf(state.location.ProfileID); profile != nil {
			subtitle = profile.Name + " · " + subtitle
		}
		a.dialogHeader(c, iconLink, a.pal.primarySoft, a.pal.primaryText, title, subtitle, nil, closeShare)
		ui.Column(c).PaddingX(18).Shrink(0).Children(func() { a.shareTabs(c) })
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(12).Padding(14, 18, 18, 18).Children(func() { a.shareBody(c) })
		})
	})
}

// shareTabs chooses between a temporary and a public link.
func (a *App) shareTabs(c *ui.Context) {
	state := &a.share
	kinds := []string{"temporary", "public"}
	labels := []string{a.t("share.temporary"), a.t("share.public")}
	index := 0
	if state.mode == "public" {
		index = 1
	}
	seg := ui.SegmentedBase(c, &index, len(kinds))
	seg.Track.FillWidth().Height(segmentHeight).Radius(8).Background(a.pal.mutedFill).
		Label(a.t("menu.share")).Children(func() {
		share := 100 / float32(len(kinds))
		a.segmentThumb(c).LeftPercent(share * float32(index)).WidthPercent(share)
		for i, label := range labels {
			glyph := iconClock
			if i == 1 {
				glyph = iconLink
			}
			segment := seg.Segment(i).Absolute().LeftPercent(share * float32(i)).Top(0).
				WidthPercent(share).Height(segmentHeight).Center()
			color, weight := a.pal.muted, 500
			if i == index {
				color, weight = a.pal.text, 600
			}
			segment.Children(func() {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, glyph).FontSize(12).TextColor(color)
					ui.Text(c, label).FontSize(12).FontWeight(weight).TextColor(color)
				})
			})
		}
	})
	if seg.Track.Changed() && kinds[index] != state.mode {
		state.mode = kinds[index]
		if state.mode == "public" && !a.hasPublicURL() {
			state.links, state.err, state.busy = nil, nil, false
			return
		}
		a.buildShare()
	}
}

// hasPublicURL reports whether the shared objects' connection serves them
// at a public address.
func (a *App) hasPublicURL() bool {
	profile := a.profileOf(a.share.location.ProfileID)
	return profile != nil && profile.PublicURL != ""
}

// shareBody shows the settings of the link kind and the links made.
func (a *App) shareBody(c *ui.Context) {
	state := &a.share
	if state.mode == "public" {
		if !a.hasPublicURL() {
			ui.Row(c).Gap(8).Padding(9, 12).Radius(9).AlignItems(ui.Center).
				Background(a.pal.mutedFill).Children(func() {
				ui.Icon(c, iconAlertTriangle).FontSize(13).TextColor(a.pal.muted)
				ui.Text(c, a.t("share.noPublic")).FontSize(12.5).TextColor(a.pal.muted).Grow(1).MinWidth(0)
				if a.secondaryButton(c, nil, a.t("states.editConnection")).Height(26).Clicked() {
					id := state.location.ProfileID
					a.share = shareState{}
					a.openConnect(id)
				}
			})
			return
		}
		a.notice(c, a.pal.warningBg, a.pal.warningText, a.t("share.publicWarn"))
	} else {
		a.shareExpiry(c)
	}
	switch {
	case state.busy:
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Size(12, 12).Label(a.t("share.preparing"))
			ui.Text(c, a.t("share.preparing")).FontSize(12.5).TextColor(a.pal.muted)
		})
		return
	case state.err != nil:
		a.notice(c, a.pal.destructiveBg, a.pal.destructive, a.errorText(state.err))
		return
	}
	multi := len(state.links) > 1 || len(state.entries) == 1 && state.entries[0].folder
	if multi {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			text := a.tf("share.multi", "count", len(state.links))
			if len(state.entries) == 1 && state.entries[0].folder {
				text = a.tf("share.folder", "count", len(state.links))
			}
			ui.Text(c, text).FontSize(12).TextColor(a.pal.muted).Grow(1)
			if a.secondaryButton(c, iconCopy, a.t("common.copyAll")).Height(26).Clicked() {
				a.copyText(c, joinLines(linkURLs(state.links)))
			}
		})
	}
	for i := range state.links {
		if i == shareLinksShown {
			break
		}
		a.shareRow(c, &state.links[i], multi)
	}
	if len(state.links) > shareLinksShown {
		ui.Text(c, a.tf("share.more", "count", len(state.links)-shareLinksShown)).
			FontSize(12).TextColor(a.pal.muted)
	}
	note := a.t("share.footTemporary")
	if state.mode == "public" {
		note = a.t("share.footPublic")
	}
	ui.Text(c, note).FontSize(11.5).TextColor(a.pal.faint)
}

// shareExpiry chooses how long a temporary link works.
func (a *App) shareExpiry(c *ui.Context) {
	state := &a.share
	labels := []string{a.t("share.d15m"), a.t("share.d1h"), a.t("share.d1d"), a.t("share.d7d")}
	index := -1
	for i, seconds := range shareDurations {
		if seconds == state.seconds {
			index = i
		}
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.t("share.validFor")).FontSize(12).TextColor(a.pal.muted)
		if a.segments(c, &index, a.t("share.validFor"), labels) && index >= 0 &&
			shareDurations[index] != state.seconds {
			state.seconds = shareDurations[index]
			a.buildShare()
		}
		ui.Spacer(c)
		if len(state.links) > 0 && state.links[0].expires != "" {
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconClock).FontSize(11).TextColor(a.pal.muted)
				ui.Text(c, a.tf("share.expiresAt", "time", state.links[0].expires)).FontSize(12).
					FontFeatures("tnum").TextColor(a.pal.muted)
			})
		}
	})
}

// shareRow is one link with its copy button. A share of many objects names
// the object of each link.
func (a *App) shareRow(c *ui.Context, link *shareLink, named bool) {
	ui.Column(c).Gap(4).Padding(8, 8, 8, 12).Radius(10).
		Background(a.pal.sidebar).Border(1, a.pal.border).Children(func() {
		if named {
			ui.Text(c, link.entry.key).FontSize(11.5).FontWeight(600).SingleLine()
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if link.err != nil {
				ui.Text(c, a.errorText(link.err)).FontSize(11.5).TextColor(a.pal.destructive).
					Grow(1).MinWidth(0)
				return
			}
			ui.Text(c, link.url).FontSize(11.5).Font("monospace").TextColor(a.pal.muted).
				SingleLine().Selectable().Grow(1).MinWidth(0)
			if a.primaryAction(c, iconCopy, a.t("common.copy"), false).Height(26).PaddingX(10).Clicked() {
				a.copyText(c, link.url)
			}
		})
		if link.url != "" && len(link.headers) > 0 {
			ui.Text(c, a.t("share.headers")).FontSize(11).TextColor(a.pal.faint).Margin(4, 0, 0, 0)
			for _, name := range sortedKeys(link.headers) {
				ui.Text(c, name+": "+link.headers[name]).FontSize(11).Font("monospace").
					TextColor(a.pal.muted).SingleLine().Selectable()
			}
		}
	})
}

func linkURLs(links []shareLink) []string {
	out := make([]string, 0, len(links))
	for _, link := range links {
		if link.url != "" {
			out = append(out, link.url)
		}
	}
	return out
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}
