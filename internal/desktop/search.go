package desktop

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// searchPaletteState is the ⌘F palette: it jumps to a connection, a bucket
// or a folder.
type searchPaletteState struct {
	open    bool
	query   string
	matches []paletteMatch
	chosen  int
	focus   ui.Handle
}

// paletteMatch is one jump target.
type paletteMatch struct {
	label    string
	hint     string
	glyph    *ui.SVG
	location Location
}

// openPalette shows the palette.
func (a *App) openPalette() {
	a.search.open = true
	a.search.query = ""
	a.search.matches = nil
	a.search.chosen = 0
}

// paletteMatches builds the jump targets for the typed text.
func (a *App) paletteMatches() []paletteMatch {
	query := strings.ToLower(strings.TrimSpace(a.search.query))
	if query == "" {
		return nil
	}
	out := []paletteMatch{}
	for _, profile := range a.profiles {
		if profile.Bucket != "" {
			if strings.Contains(strings.ToLower(profile.Name), query) ||
				strings.Contains(strings.ToLower(profile.Bucket), query) {
				out = append(out, paletteMatch{
					label: profile.Name, hint: profile.Bucket, glyph: iconDatabase,
					location: Location{ProfileID: profile.ID, Bucket: profile.Bucket},
				})
			}
			continue
		}
		state := a.buckets[profile.ID]
		if state == nil {
			a.ensureBuckets(profile.ID)
			continue
		}
		for _, bucket := range state.buckets {
			if strings.Contains(strings.ToLower(bucket.Name), query) ||
				strings.Contains(strings.ToLower(profile.Name), query) {
				out = append(out, paletteMatch{
					label: bucket.Name, hint: profile.Name, glyph: iconDatabase,
					location: Location{ProfileID: profile.ID, Bucket: bucket.Name},
				})
			}
		}
	}
	if a.location != nil {
		hint := a.location.Prefix
		if hint == "" {
			hint = a.location.Bucket
		}
		for _, row := range a.browse.shown {
			if !row.folder || !strings.Contains(strings.ToLower(row.name), query) {
				continue
			}
			out = append(out, paletteMatch{
				label: row.name, hint: hint, glyph: iconFolder,
				location: Location{
					ProfileID: a.location.ProfileID, Bucket: a.location.Bucket, Prefix: row.key,
				},
			})
		}
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// searchOverlay draws the jump palette.
func (a *App) searchOverlay(c *ui.Context) {
	state := &a.search
	if !state.open {
		return
	}
	state.matches = a.paletteMatches()
	if state.chosen >= len(state.matches) {
		state.chosen = 0
	}
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Start).Padding(0, 0, 120, 0).
			Background(a.pal.scrim)
		backdrop.Children(func() {
			panel := ui.Column(c).Width(560).Margin(120, 0, 0, 0).Radius(14).
				Background(a.pal.popover).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.24))
			panel.Clip()
			panel.Children(func() {
				ui.Row(c).Padding(10, 12).Gap(10).AlignItems(ui.Center).
					BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
					ui.Icon(c, iconSearch).FontSize(16).TextColor(a.pal.faint)
					input := ui.TextInput(c, &state.query).
						Placeholder(a.t("browser.searchPlaceholder")).AutoFocus().
						Border(0, ui.Transparent).Grow(1)
					input.Bind(&state.focus)
				})
				ui.Scroll(c).MaxHeight(340).Padding(6).Children(func() {
					ui.Column(c).Gap(1).Children(func() {
						if strings.TrimSpace(state.query) == "" {
							ui.Text(c, a.t("search.hint")).FontSize(11.5).
								TextColor(a.pal.faint).Padding(10)
							return
						}
						if len(state.matches) == 0 {
							ui.Text(c, a.t("search.noJump")).FontSize(12).
								TextColor(a.pal.muted).Padding(10)
							return
						}
						ui.Text(c, a.t("search.jumpTo")).FontSize(10.5).
							TextColor(a.pal.faint).Padding(6, 10)
						for i := range state.matches {
							a.paletteRow(c, i)
						}
					})
				})
				ui.Row(c).Height(28).PaddingX(12).Gap(12).AlignItems(ui.Center).
					BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
					for _, hint := range []string{
						a.t("search.keySelect"), a.t("search.keyOpen"), a.t("search.keyClose"),
					} {
						ui.Text(c, hint).FontSize(10.5).TextColor(a.pal.faint)
					}
				})
			})
			if backdrop.PressedOutside() || backdrop.OverlayShortcut(0, ui.KeyEscape) {
				state.open = false
			}
		})
	})
}

// paletteRow is one jump target.
func (a *App) paletteRow(c *ui.Context, index int) {
	state := &a.search
	match := state.matches[index]
	row := ui.Row(c).Padding(8, 10).Gap(10).Radius(6).AlignItems(ui.Center)
	if index == state.chosen {
		row.Background(a.pal.primary)
	}
	row.Children(func() {
		color := a.pal.muted
		textColor := a.pal.text
		hintColor := a.pal.faint
		if index == state.chosen {
			color = a.pal.onPrimary
			textColor = a.pal.onPrimary
			hintColor = a.pal.onPrimary
		}
		ui.Icon(c, match.glyph).FontSize(15).TextColor(color)
		ui.Text(c, match.label).FontSize(12.5).SingleLine().TextColor(textColor)
		ui.Spacer(c)
		ui.Text(c, match.hint).FontSize(11).TextColor(hintColor).SingleLine()
	})
	if row.Hovered() {
		state.chosen = index
	}
	if row.Clicked() {
		state.open = false
		a.navigate(match.location)
	}
}

// paletteShortcuts handles the keys the palette takes while it is open.
func (a *App) paletteShortcuts(c *ui.Context) {
	state := &a.search
	if !state.open {
		return
	}
	if c.Shortcut(0, ui.KeyDown) {
		if state.chosen+1 < len(state.matches) {
			state.chosen++
		}
	}
	if c.Shortcut(0, ui.KeyUp) {
		if state.chosen > 0 {
			state.chosen--
		}
	}
	if c.Shortcut(0, ui.KeyEnter) && len(state.matches) > 0 {
		target := state.matches[state.chosen]
		state.open = false
		a.navigate(target.location)
	}
}
