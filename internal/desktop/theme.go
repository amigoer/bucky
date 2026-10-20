package desktop

import "github.com/egoist/mygo/ui"

// accentTone is the hue and chroma of one accent color. The lightness is
// fixed per appearance, so every accent keeps the same contrast.
type accentTone struct {
	hue    float32
	chroma float32
}

var accentTones = map[string]accentTone{
	"blue":     {hue: 255, chroma: 0.165},
	"indigo":   {hue: 275, chroma: 0.18},
	"purple":   {hue: 305, chroma: 0.17},
	"pink":     {hue: 355, chroma: 0.17},
	"orange":   {hue: 48, chroma: 0.15},
	"green":    {hue: 152, chroma: 0.13},
	"graphite": {hue: 260, chroma: 0.012},
}

func accentToneOf(name string) accentTone {
	if tone, ok := accentTones[name]; ok {
		return tone
	}
	return accentTones["blue"]
}

// kindPalette is the background and foreground of one file kind, as the
// tiles and the usage chart show it.
type kindPalette struct {
	background ui.Color
	foreground ui.Color
}

// palette holds every color the interface draws with, built once per
// appearance and accent change.
type palette struct {
	dark bool

	background    ui.Color
	sidebar       ui.Color
	text          ui.Color
	muted         ui.Color
	faint         ui.Color
	border        ui.Color
	borderStrong  ui.Color
	hover         ui.Color
	mutedFill     ui.Color
	field         ui.Color
	input         ui.Color
	card          ui.Color
	popover       ui.Color
	primary       ui.Color
	primaryText   ui.Color
	primarySoft   ui.Color
	primaryTint   ui.Color
	primaryLine   ui.Color
	onPrimary     ui.Color
	destructive   ui.Color
	destructiveBg ui.Color
	warning       ui.Color
	warningText   ui.Color
	warningBg     ui.Color
	success       ui.Color
	successText   ui.Color
	successBg     ui.Color
	scrim         ui.Color
	hud           ui.Color
	tile          ui.Color
	logoFrom      ui.Color
	logoTo        ui.Color
	primaryHover  ui.Color
	thumb         ui.Color
	thumbShadow   ui.Color
	// stripe is drawn in place of the tile, not over it, so it includes it.
	stripe       ui.Color
	folderFill   ui.Color
	folderStroke ui.Color
	// button, buttonRing and buttonShadow are a secondary button's face.
	button       ui.Color
	buttonRing   ui.Color
	buttonShadow ui.Color
	// onPrimaryMuted is secondary text on the accent, as a chosen row's.
	onPrimaryMuted ui.Color

	kinds  map[string]kindPalette
	folder ui.Color
}

func newPalette(dark bool, accent string) *palette {
	tone := accentToneOf(accent)
	lightness := float32(0.56)
	if dark {
		lightness = 0.63
	}
	primary := ui.Oklch(lightness, tone.chroma, tone.hue)

	p := &palette{
		dark:           dark,
		primary:        primary,
		primaryText:    ui.Oklch(0.5, tone.chroma, tone.hue),
		primarySoft:    ui.Oklch(lightness, tone.chroma, tone.hue).Alpha(0.13),
		primaryTint:    ui.Oklch(lightness, tone.chroma, tone.hue).Alpha(0.06),
		primaryLine:    ui.Oklch(lightness, tone.chroma, tone.hue).Alpha(0.3),
		onPrimary:      ui.Hex("#ffffff"),
		destructive:    ui.Oklch(0.56, 0.2, 27),
		destructiveBg:  ui.Oklch(0.58, 0.2, 27).Alpha(0.1),
		warning:        ui.Oklch(0.75, 0.13, 75),
		warningText:    ui.Oklch(0.46, 0.1, 70),
		warningBg:      ui.Oklch(0.86, 0.1, 85).Alpha(0.5),
		success:        ui.Oklch(0.62, 0.14, 150),
		successText:    ui.Oklch(0.46, 0.12, 150),
		successBg:      ui.Oklch(0.62, 0.14, 150).Alpha(0.14),
		hud:            ui.RGBA(30, 30, 32, 0.94),
		folder:         ui.Oklch(0.8, tone.chroma*0.55, tone.hue),
		logoFrom:       ui.Oklch(0.66, tone.chroma, tone.hue),
		logoTo:         ui.Oklch(0.52, tone.chroma, tone.hue),
		primaryHover:   ui.Oklch(lightness+0.04, tone.chroma, tone.hue),
		thumb:          ui.Hex("#ffffff"),
		thumbShadow:    ui.RGBA(0, 0, 0, 0.14),
		folderFill:     ui.Oklch(0.86, 0.07, 70),
		folderStroke:   ui.Oklch(0.62, 0.09, 60),
		onPrimaryMuted: ui.RGBA(255, 255, 255, 0.82),
		button:         ui.Hex("#ffffff"),
		buttonRing:     ui.RGBA(0, 0, 0, 0.16),
		buttonShadow:   ui.RGBA(0, 0, 0, 0.05),
		kinds:          map[string]kindPalette{},
	}
	if dark {
		p.background = ui.Hex("#1e1e20")
		p.sidebar = ui.Hex("#252528")
		p.text = ui.Hex("#ececee")
		p.muted = ui.Hex("#9c9ca2")
		p.faint = ui.Hex("#6d6d73")
		p.border = ui.RGBA(255, 255, 255, 0.08)
		p.borderStrong = ui.RGBA(255, 255, 255, 0.15)
		p.hover = ui.RGBA(255, 255, 255, 0.05)
		p.mutedFill = ui.RGBA(255, 255, 255, 0.08)
		p.field = ui.RGBA(255, 255, 255, 0.07)
		p.input = ui.RGBA(255, 255, 255, 0.08)
		p.card = ui.Hex("#1e1e20")
		p.popover = ui.Hex("#2b2b2e")
		p.scrim = ui.RGBA(0, 0, 0, 0.42)
		p.tile = ui.RGBA(255, 255, 255, 0.05)
		p.thumb = ui.RGBA(255, 255, 255, 0.18)
		p.thumbShadow = ui.Transparent
		p.stripe = ui.RGBA(255, 255, 255, 0.1)
		p.folderFill = ui.Oklch(0.5, 0.06, 65)
		p.folderStroke = ui.Oklch(0.72, 0.07, 65)
		p.button = ui.RGBA(255, 255, 255, 0.1)
		p.buttonRing = ui.RGBA(255, 255, 255, 0.14)
		p.buttonShadow = ui.Transparent
		p.warningBg = ui.Oklch(0.5, 0.1, 75).Alpha(0.3)
		p.destructiveBg = ui.Oklch(0.58, 0.2, 27).Alpha(0.18)
		// Secondary colors have to read on a dark surface, so they are
		// lighter than their light-theme counterparts.
		p.warningText = ui.Oklch(0.82, 0.12, 75)
		p.successText = ui.Oklch(0.78, 0.13, 150)
		p.primaryText = ui.Oklch(0.78, tone.chroma*0.9, tone.hue)
	} else {
		p.background = ui.Hex("#ffffff")
		p.sidebar = ui.Hex("#f4f4f6")
		p.text = ui.Hex("#1d1d1f")
		p.muted = ui.Hex("#6e6e73")
		p.faint = ui.Hex("#a1a1a6")
		p.border = ui.RGBA(0, 0, 0, 0.07)
		p.borderStrong = ui.RGBA(0, 0, 0, 0.12)
		p.hover = ui.RGBA(0, 0, 0, 0.04)
		p.mutedFill = ui.RGBA(0, 0, 0, 0.06)
		p.field = ui.RGBA(0, 0, 0, 0.05)
		p.input = ui.RGBA(0, 0, 0, 0.07)
		p.card = ui.Hex("#ffffff")
		p.popover = ui.Hex("#ffffff")
		p.scrim = ui.RGBA(0, 0, 0, 0.14)
		p.tile = ui.Hex("#f1f1f3")
		p.stripe = ui.RGBA(0, 0, 0, 0.055).Over(p.tile)
	}

	// Kind colors keep a fixed hue per kind and follow the appearance in
	// lightness, as the interface did.
	tint := func(hue, chroma float32) kindPalette {
		if dark {
			return kindPalette{
				background: ui.Oklch(0.33, chroma*0.25, hue),
				foreground: ui.Oklch(0.8, chroma, hue),
			}
		}
		return kindPalette{
			background: ui.Oklch(0.95, chroma*0.25, hue),
			foreground: ui.Oklch(0.52, chroma, hue),
		}
	}
	p.kinds["folder"] = tint(70, 0.09)
	p.kinds["image"] = tint(42, 0.12)
	p.kinds["video"] = tint(320, 0.12)
	p.kinds["audio"] = tint(200, 0.12)
	p.kinds["pdf"] = tint(25, 0.12)
	p.kinds["json"] = tint(150, 0.12)
	p.kinds["code"] = tint(150, 0.12)
	p.kinds["text"] = tint(250, 0.12)
	p.kinds["font"] = tint(30, 0.08)
	p.kinds["archive"] = tint(262, 0.01)
	p.kinds["other"] = tint(262, 0.01)
	return p
}

// kindPaletteOf returns the colors of a kind, falling back to the neutral
// one for kinds the palette does not name.
func (p *palette) kindPaletteOf(kind string) kindPalette {
	if entry, ok := p.kinds[kind]; ok {
		return entry
	}
	return p.kinds["other"]
}

// widgetTheme turns the palette into the theme the MyGo widgets draw with.
func (p *palette) widgetTheme() *ui.Theme {
	t := ui.LightTheme()
	if p.dark {
		t = ui.DarkTheme()
	}
	t.Background = p.background
	t.Surface = p.field
	t.SurfaceHover = p.mutedFill
	t.SurfacePressed = p.borderStrong
	t.Border = p.borderStrong
	t.Text = p.text
	t.TextMuted = p.muted
	t.Accent = p.primary
	t.AccentHover = p.primary
	t.AccentPressed = p.primaryText
	t.AccentText = p.onPrimary
	t.Danger = p.destructive
	t.Warning = p.warningText
	t.Success = p.successText
	t.Selection = p.primarySoft
	t.Focus = p.primaryLine
	t.Inverse = p.hud
	t.InverseText = ui.Hex("#f4f4f6")
	if p.dark {
		t.InverseText = ui.Hex("#ececee")
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.28)
	} else {
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.22)
	}
	t.ScrollbarWidth = 8
	t.Radius = 8
	t.Spacing = 4
	return t
}
