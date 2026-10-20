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

// shareOverlay draws the share dialog over the window.
func (a *App) shareOverlay(c *ui.Context) {
	state := &a.share
	if !state.open {
		return
	}
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Background(a.pal.scrim)
		backdrop.Children(func() {
			panel := ui.Column(c).Width(620).MaxHeightPercent(84).Radius(16).
				Background(a.pal.background).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.22))
			panel.Clip()
			panel.Children(func() {
				a.shareHeader(c)
				a.shareOptions(c)
				ui.Column(c).Grow(1).Children(func() { a.shareBody(c) })
				a.shareFooter(c)
			})
			if backdrop.OverlayShortcut(0, ui.KeyEscape) {
				a.share = shareState{}
			}
		})
	})
}

func (a *App) shareHeader(c *ui.Context) {
	state := &a.share
	ui.Row(c).Height(52).PaddingX(18).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		ui.Icon(c, iconShare).FontSize(18).TextColor(a.pal.primary)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, a.t("menu.share")).FontSize(14).Bold()
			ui.Text(c, a.tn("share.objects", len(state.entries), "count", len(state.entries))).
				FontSize(11.5).TextColor(a.pal.muted)
		})
		if a.iconAction(c, iconX, a.t("common.close")) {
			a.share = shareState{}
		}
	})
}

// shareOptions chooses the link kind and its lifetime.
func (a *App) shareOptions(c *ui.Context) {
	state := &a.share
	profile := a.currentProfile()
	hasPublic := profile != nil && profile.PublicURL != ""
	ui.Column(c).Gap(10).Padding(14, 18).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		kinds := []string{"temporary"}
		labels := []string{a.t("share.temporary")}
		if hasPublic {
			kinds = append(kinds, "public")
			labels = append(labels, a.t("share.public"))
		}
		index := 0
		for i, kind := range kinds {
			if kind == state.mode {
				index = i
			}
		}
		seg := ui.Segmented(c, &index, labels...).Label(a.t("menu.share")).Width(260)
		if seg.Changed() && index >= 0 && index < len(kinds) && kinds[index] != state.mode {
			state.mode = kinds[index]
			a.buildShare()
		}
		if state.mode == "temporary" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("share.validFor")).FontSize(12).TextColor(a.pal.muted)
				durationLabels := []string{
					a.t("share.d15m"), a.t("share.d1h"), a.t("share.d1d"), a.t("share.d7d"),
				}
				current := durationLabels[1]
				for i, seconds := range shareDurations {
					if seconds == state.seconds {
						current = durationLabels[i]
					}
				}
				select_ := ui.Select(c, &current, durationLabels).
					Label(a.t("share.validFor")).Width(120)
				if select_.Changed() {
					for i, label := range durationLabels {
						if label == current && shareDurations[i] != state.seconds {
							state.seconds = shareDurations[i]
							a.buildShare()
							return
						}
					}
				}
			})
		} else {
			ui.Text(c, a.t("share.publicWarn")).FontSize(11.5).TextColor(a.pal.warningText)
		}
		if !hasPublic {
			ui.Text(c, a.t("share.noPublic")).FontSize(11.5).TextColor(a.pal.faint)
		}
	})
}

// shareBody lists the links made for the choice.
func (a *App) shareBody(c *ui.Context) {
	state := &a.share
	switch {
	case state.busy:
		ui.Column(c).Height(200).Center().Gap(10).Children(func() {
			ui.Spinner(c).Label(a.t("share.preparing"))
			ui.Text(c, a.t("share.preparing")).FontSize(12.5).TextColor(a.pal.muted)
		})
	case state.err != nil:
		ui.Column(c).Height(200).Center().Padding(24).Children(func() {
			ui.Text(c, a.errorText(state.err)).FontSize(12.5).TextColor(a.pal.muted).
				TextAlign(ui.Center)
		})
	default:
		ui.Scroll(c).MaxHeight(320).Padding(12, 18).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				for i := range state.links {
					a.shareRow(c, &state.links[i])
				}
			})
		})
	}
}

// shareRow is one link with its copy button.
func (a *App) shareRow(c *ui.Context, link *shareLink) {
	ui.Column(c).Gap(6).Padding(10, 12).Radius(10).
		Background(a.pal.card).Border(1, a.pal.border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconLink).FontSize(14).TextColor(a.pal.muted)
			ui.Text(c, link.entry.name).FontSize(12.5).Bold().SingleLine().Grow(1)
			if link.err != nil {
				ui.Text(c, a.errorText(link.err)).FontSize(11).TextColor(a.pal.destructive)
			} else {
				copy := ui.Button(c, a.t("common.copy")).FontSize(11.5)
				if copy.Clicked() {
					a.copyText(c, link.url)
				}
			}
		})
		if link.url != "" {
			ui.Text(c, link.url).FontSize(11).Font("monospace").
				TextColor(a.pal.muted).SingleLine().Selectable()
			if len(link.headers) > 0 {
				ui.Text(c, a.t("share.headers")).FontSize(10.5).TextColor(a.pal.faint)
				for _, name := range sortedKeys(link.headers) {
					ui.Text(c, name+": "+link.headers[name]).FontSize(10.5).
						TextColor(a.pal.muted).SingleLine().Selectable()
				}
			}
			if link.expires != "" {
				ui.Text(c, a.tf("share.expiresAt", "time", link.expires)).FontSize(10.5).
					TextColor(a.pal.faint)
			}
		}
	})
}

// shareFooter explains how the links were made and copies them all.
func (a *App) shareFooter(c *ui.Context) {
	state := &a.share
	ui.Column(c).Gap(8).Padding(12, 18).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		note := a.t("share.footTemporary")
		if state.mode == "public" {
			note = a.t("share.footPublic")
		}
		ui.Text(c, note).FontSize(11).TextColor(a.pal.faint)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if len(state.links) > 1 {
				if ui.Button(c, a.t("common.copyAll")).Clicked() {
					a.copyText(c, joinLines(linkURLs(state.links)))
				}
			}
			if ui.PrimaryButton(c, a.t("common.done")).Clicked() {
				a.share = shareState{}
			}
		})
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
