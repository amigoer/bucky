package desktop

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Version of the desktop application, as the About section shows it.
const Version = "0.1.0"

// mcpSnippet is the client configuration the integrations section shows.
const mcpSnippet = `{
  "mcpServers": {
    "bucky": { "command": "bucky", "args": ["mcp"] }
  }
}`

const cliSnippet = `bucky ls r2://bucket/prefix --profile "Acme assets"`

const skillSnippet = "bucky skill print"

var mcpTools = []string{
	"profiles", "buckets", "list", "stat", "find", "read_text", "presign",
	"usage", "transfers", "get_plan", "create_plan",
}

// settingsPage draws every setting on one page, a section under another.
func (a *App) settingsPage(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).MaxWidth(700).Padding(0, 28, 56, 28).Children(func() {
			a.settingsHeading(c, a.t("settings.sections.appearance"))
			a.themeChoice(c)
			a.settingsCard(c, func() {
				a.settingsRow(c, a.t("settings.accent"), a.t("settings.accentDesc"), func() {
					ui.Row(c).Gap(8).Children(func() {
						for _, name := range accentNames {
							a.accentSwatch(c, name)
						}
					})
				})
			}).Margin(14, 0, 0, 0)
			a.settingsHeading(c, a.t("settings.language"))
			a.languageChoice(c)
			a.settingsHeading(c, a.t("settings.sections.transfers"))
			a.settingsCard(c, func() {
				a.settingsRow(c, a.t("settings.rateLimit"), a.t("settings.rateLimitHint"), func() {
					a.rateStepper(c, 28)
				})
				a.settingsDivider(c)
				a.settingsRow(c, a.t("settings.concurrency"), a.t("settings.concurrencyHint"), func() {
					a.concurrencySwitch(c)
				})
			})
			a.settingsHeading(c, a.t("usage.title"))
			a.settingsCard(c, func() {
				ui.Row(c).Gap(12).Padding(14, 16).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.t("settings.usageHint")).FontSize(12.5).TextColor(a.pal.muted).Grow(1)
					if a.secondaryButton(c, iconPieChart, a.t("settings.openUsage")).Clicked() {
						a.page = pageUsage
						a.ensureUsage()
					}
				})
			})
			a.settingsHeading(c, "CLI · MCP · Agent Skill")
			a.agentCard(c)
			a.settingsHeading(c, a.t("settings.cloudflare"))
			a.cloudflareCard(c)
			a.settingsHeading(c, a.t("settings.sections.shortcuts"))
			a.shortcutGrid(c)
			a.settingsHeading(c, a.t("settings.privacy"))
			a.settingsCard(c, func() {
				a.promiseRow(c, iconLock, a.t("settings.privacyDesc"))
				a.settingsDivider(c)
				a.promiseRow(c, iconShieldCheck, a.t("settings.safetyDesc"))
			})
			a.settingsHeading(c, a.t("settings.sections.about"))
			a.aboutCard(c)
		})
	})
}

// settingsHeading names a section of the settings page.
func (a *App) settingsHeading(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(a.pal.muted).Padding(26, 0, 10, 0)
}

// settingsCard is a bordered group of settings.
func (a *App) settingsCard(c *ui.Context, rows func()) ui.Element {
	card := ui.Column(c).Radius(12).Border(1, a.pal.border)
	card.Children(rows)
	return card
}

func (a *App) settingsDivider(c *ui.Context) {
	ui.Box(c).Height(1).Background(a.pal.border)
}

// settingsRow is a label, its explanation and the control.
func (a *App) settingsRow(c *ui.Context, title, description string, control func()) {
	ui.Row(c).Padding(14, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13).FontWeight(500)
			if description != "" {
				ui.Text(c, description).FontSize(11.5).TextColor(a.pal.muted)
			}
		})
		control()
	})
}

// A theme preview is a small window: a sidebar and a few tiles.
const (
	themePreviewWidth  = 200
	themePreviewHeight = 125
)

// themeChoice picks the theme from three previews.
func (a *App) themeChoice(c *ui.Context) {
	light := themeSample{window: ui.Hex("#ffffff"), sidebar: ui.Hex("#f0f0f2"), line: ui.RGBA(0, 0, 0, 0.1)}
	dark := themeSample{window: ui.Hex("#1e1e20"), sidebar: ui.Hex("#2a2a2d"), line: ui.RGBA(255, 255, 255, 0.14)}
	options := []struct {
		name, label string
		glyph       *ui.SVG
		left, right themeSample
	}{
		{themeLight, a.t("settings.light"), iconSun, light, light},
		{themeDark, a.t("settings.dark"), iconMoon, dark, dark},
		{themeSystem, a.t("settings.system"), iconMonitor, light, dark},
	}
	ui.Row(c).Gap(14).Wrap().Children(func() {
		for _, option := range options {
			chosen := option.name == a.prefs.Theme
			item := ui.Column(c).Gap(9).Cursor(ui.CursorPointer).Label(option.label)
			item.Children(func() {
				preview := ui.Box(c).Size(themePreviewWidth, themePreviewHeight).Radius(11)
				if chosen {
					preview.Shadow(0, 0, 0, 6, a.pal.primarySoft).Shadow(0, 0, 0, 2, a.pal.primary)
				} else {
					preview.Shadow(0, 0, 0, 1, a.pal.borderStrong)
				}
				preview.Children(func() {
					ui.Box(c).Fill().Radius(11).Clip().Children(func() {
						a.themeSample(c, option.left)
						if option.right != option.left {
							// The system preview is light on the left and dark on the right.
							half := float32(themePreviewWidth / 2)
							ui.Box(c).Absolute().Left(half).Top(0).Size(half, themePreviewHeight).Clip().
								Children(func() {
									ui.Box(c).Absolute().Left(-half).Top(0).Children(func() {
										a.themeSample(c, option.right)
									})
								})
						}
					})
					if chosen {
						ui.Box(c).Absolute().Right(7).Top(7).Size(18, 18).Radius(9).Center().
							Background(a.pal.primary).Children(func() {
							ui.Icon(c, iconCheck).FontSize(11).TextColor(a.pal.onPrimary)
						})
					}
				})
				color, weight := a.pal.muted, 500
				if chosen {
					color, weight = a.pal.primaryText, 600
				}
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, option.glyph).FontSize(13).TextColor(color)
					ui.Text(c, option.label).FontSize(12.5).FontWeight(weight).TextColor(color)
				})
			})
			if item.Clicked() && !chosen {
				a.prefs.Theme = option.name
				a.savePrefs()
				a.refreshPalette()
			}
		}
	})
}

// themeSample is the colors of one theme preview.
type themeSample struct {
	window, sidebar, line ui.Color
}

// themeSample draws a window in a theme's colors, with the accent in it.
func (a *App) themeSample(c *ui.Context, sample themeSample) {
	ui.Row(c).Size(themePreviewWidth, themePreviewHeight).AlignItems(ui.Stretch).
		Background(sample.window).Children(func() {
		ui.Column(c).Width(themePreviewWidth*0.3).Padding(9, 7).Gap(5).
			Background(sample.sidebar).Children(func() {
			for i, width := range []float32{70, 58, 64} {
				color := sample.line
				if i == 1 {
					color = a.pal.primary
				}
				ui.Box(c).Height(5).WidthPercent(width).Radius(3).Background(color)
			}
		})
		ui.Row(c).Grow(1).Padding(10).Gap(6).AlignItems(ui.Start).Children(func() {
			for i := range 3 {
				tile := ui.Box(c).Grow(1).Height(30).Radius(3).Background(sample.line)
				if i == 2 {
					tile.Background(a.pal.primary).Opacity(0.75)
				}
			}
		})
	})
}

// accentSwatch chooses one accent color.
func (a *App) accentSwatch(c *ui.Context, name string) {
	tone := accentToneOf(name)
	color := ui.Oklch(0.56, tone.chroma, tone.hue)
	button := ui.Box(c).Size(22, 22).Radius(11).Background(color).Cursor(ui.CursorPointer)
	button.Label(a.t("settings.accents." + name)).Tooltip(a.t("settings.accents." + name))
	if name == a.prefs.Accent {
		button.Shadow(0, 0, 0, 4, color).Shadow(0, 0, 0, 2, a.pal.background)
	} else {
		button.InsetShadow(0, 0, 0, 1, ui.RGBA(0, 0, 0, 0.12))
	}
	if button.Clicked() {
		a.prefs.Accent = name
		a.savePrefs()
		a.refreshPalette()
	}
}

// languageChoice picks the interface language. Each option is written in
// its own language.
func (a *App) languageChoice(c *ui.Context) {
	options := []struct{ lang, label, glyph, sample string }{
		{langChinese, "简体中文", "文", "上传 · 分享 · 删除"},
		{langEnglish, "English", "Aa", "Upload · Share · Delete"},
	}
	ui.Row(c).Gap(12).Children(func() {
		for _, option := range options {
			chosen := option.lang == a.prefs.Language
			card := ui.Row(c).Grow(1).Gap(12).Padding(12, 14).Radius(11).AlignItems(ui.Center).
				Cursor(ui.CursorPointer).Label(option.label)
			if chosen {
				card.Shadow(0, 0, 0, 6, a.pal.primarySoft).Shadow(0, 0, 0, 2, a.pal.primary)
			} else {
				card.Border(1, a.pal.borderStrong)
			}
			card.Children(func() {
				tile, ink := a.pal.mutedFill, a.pal.muted
				if chosen {
					tile, ink = a.pal.primary, a.pal.onPrimary
				}
				ui.Box(c).Size(38, 38).Radius(10).Center().Shrink(0).Background(tile).Children(func() {
					ui.Text(c, option.glyph).FontSize(16).FontWeight(650).TextColor(ink)
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, option.label).FontSize(13).FontWeight(600).SingleLine()
					ui.Text(c, option.sample).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
				})
				ring := a.pal.borderStrong
				if chosen {
					ring = a.pal.primary
				}
				ui.Box(c).Size(16, 16).Radius(8).Center().Shrink(0).Border(1.5, ring).Children(func() {
					if chosen {
						ui.Box(c).Size(8, 8).Radius(4).Background(a.pal.primary)
					}
				})
			})
			if card.Clicked() {
				a.setLanguage(option.lang)
			}
		}
	})
}

// agentCard shows how the CLI, MCP clients and agents reach the same core.
func (a *App) agentCard(c *ui.Context) {
	a.settingsCard(c, func() {
		a.agentSection(c, iconSquareTerminal, "MCP", a.t("settings.mcpDesc"), mcpSnippet, func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Wrap().Children(func() {
				ui.Text(c, a.t("settings.tools")).FontSize(11.5).TextColor(a.pal.muted)
				for _, tool := range mcpTools {
					ui.Text(c, tool).FontSize(11).Font("monospace").Height(20).PaddingX(7).Radius(5).
						Background(a.pal.mutedFill)
				}
			})
		})
		a.settingsDivider(c)
		a.agentSection(c, iconSquareTerminal, "CLI", a.t("settings.cliDesc"), cliSnippet, nil)
		a.settingsDivider(c)
		a.agentSection(c, iconFileText, "Agent Skill", a.t("settings.skillDesc"), skillSnippet, nil)
	})
}

// agentSection is one way in, with the snippet that sets it up.
func (a *App) agentSection(c *ui.Context, glyph *ui.SVG, title, description, code string, extra func()) {
	ui.Column(c).Gap(8).Padding(14, 16).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, glyph).FontSize(15).TextColor(a.pal.text)
			ui.Text(c, title).FontSize(13).FontWeight(600)
		})
		ui.Text(c, description).FontSize(12.5).LineHeight(1.6).TextColor(a.pal.muted)
		a.codeBlock(c, code)
		if extra != nil {
			extra()
		}
	})
}

// codeBlock shows a snippet with a button that copies it.
func (a *App) codeBlock(c *ui.Context, code string) {
	ui.Row(c).Gap(8).Padding(10, 10, 10, 14).Radius(9).AlignItems(ui.Start).
		Background(a.pal.sidebar).Border(1, a.pal.border).Children(func() {
		ui.Text(c, code).FontSize(12).LineHeight(1.6).Font("monospace").Grow(1).MinWidth(0).Selectable()
		if a.secondaryButton(c, iconCopy, a.t("common.copy")).Height(26).Clicked() {
			a.copyText(c, code)
		}
	})
}

// cloudflareCard holds the OAuth client settings the sign-in needs.
func (a *App) cloudflareCard(c *ui.Context) {
	a.loadSettings()
	field := func(label, hint string, value *string) {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, label).FontSize(12).FontWeight(500)
			input := ui.TextInput(c, value).Placeholder(label).Width(360)
			if input.Submitted() {
				a.saveCloudflareSettings()
			}
			ui.Text(c, hint).FontSize(11.5).TextColor(a.pal.muted)
		})
	}
	a.settingsCard(c, func() {
		ui.Column(c).Gap(12).Padding(14, 16).Children(func() {
			ui.Text(c, a.t("settings.cloudflareDesc")).FontSize(12.5).LineHeight(1.6).TextColor(a.pal.muted)
			ui.Text(c, a.t("settings.cloudflareSetup")).FontSize(12).LineHeight(1.6).TextColor(a.pal.muted)
			field(a.t("settings.cloudflareClientId"), a.t("settings.cloudflareClientIdHint"), &a.settings.clientID)
			field(a.t("settings.cloudflareRedirect"), a.t("settings.cloudflareRedirectHint"), &a.settings.redirect)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if a.primaryAction(c, nil, a.t("common.save"), false).Clicked() {
					a.saveCloudflareSettings()
				}
				if a.secondaryButton(c, iconExternalLink, a.t("settings.cloudflareDocs")).Clicked() {
					_ = mygo.Shell.OpenExternal("https://developers.cloudflare.com/fundamentals/api/get-started/create-token/")
				}
				if a.settings.notice != "" {
					ui.Text(c, a.settings.notice).FontSize(12).TextColor(a.pal.successText)
				}
			})
		})
	})
}

// loadSettings reads the Cloudflare sign-in settings once.
func (a *App) loadSettings() {
	if a.settings.loaded || a.settings.busy {
		return
	}
	a.settings.busy = true
	go func() {
		settings, err := a.svc.Settings(a.ctx)
		a.update(func() {
			a.settings.busy = false
			a.settings.loaded = true
			if err != nil {
				a.toastCurrent(a.errorText(err))
				return
			}
			a.settings.settings = settings
			a.settings.clientID = settings.CloudflareClientID
			a.settings.redirect = settings.CloudflareRedirectURI
			a.settings.scopes = settings.CloudflareScopes
		})
	}()
}

// saveCloudflareSettings stores the OAuth client settings.
func (a *App) saveCloudflareSettings() {
	input := a.settings.settings
	input.CloudflareClientID = strings.TrimSpace(a.settings.clientID)
	input.CloudflareRedirectURI = strings.TrimSpace(a.settings.redirect)
	go func() {
		saved, err := a.svc.SaveSettings(a.ctx, input)
		a.update(func() {
			if err != nil {
				a.toastCurrent(a.errorText(err))
				return
			}
			a.settings.settings = saved
			a.settings.notice = a.t("settings.cloudflareSaved")
			a.toastCurrent(a.t("settings.cloudflareSaved"))
		})
	}()
}

// shortcutGrid lists the keyboard shortcuts in two columns.
func (a *App) shortcutGrid(c *ui.Context) {
	mod := "Ctrl"
	if platformIsMac() {
		mod = "⌘"
	}
	shortcuts := []struct{ label, keys string }{
		{a.t("keys.search"), mod + "F"},
		{a.t("keys.views"), mod + "1–3"},
		{a.t("keys.open"), "↵"},
		{a.t("keys.history"), mod + "[ / ]"},
		{a.t("keys.selectAll"), mod + "A"},
		{a.t("keys.quickLook"), "Space"},
		{a.t("keys.delete"), mod + "⌫"},
		{a.t("keys.close"), "Esc"},
		{a.t("keys.quickUpload"), mod + "⌥U"},
		{a.t("keys.transfers"), mod + "⇧T"},
		{a.t("keys.settings"), mod + ","},
		{a.t("keys.sidebar"), mod + "⌃S"},
	}
	a.settingsCard(c, func() {
		for i := 0; i < len(shortcuts); i += 2 {
			line := ui.Row(c)
			if i+2 < len(shortcuts) {
				line.BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border)
			}
			line.Children(func() {
				for j := i; j < i+2 && j < len(shortcuts); j++ {
					cell := ui.Row(c).Grow(1).MinWidth(0).Padding(10, 16).Gap(12).AlignItems(ui.Center)
					if j == i {
						cell.BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border)
					}
					cell.Children(func() {
						ui.Text(c, shortcuts[j].label).FontSize(12.5).SingleLine().Grow(1).MinWidth(0)
						a.keycap(c, shortcuts[j].keys)
					})
				}
			})
		}
	})
}

// keycap draws keys as a key on a keyboard.
func (a *App) keycap(c *ui.Context, keys string) {
	ui.Text(c, keys).FontSize(11).Font("monospace").TextColor(a.pal.muted).
		Height(20).MinWidth(22).PaddingX(6).Radius(5).TextAlign(ui.Center).
		Background(a.pal.sidebar).Border(1, a.pal.borderStrong).Shadow(0, 1, 0, 0, a.pal.borderStrong)
}

// promiseRow is one promise the application makes about the user's data.
func (a *App) promiseRow(c *ui.Context, glyph *ui.SVG, text string) {
	ui.Row(c).Gap(12).Padding(14, 16).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(32, 32).Radius(9).Center().Shrink(0).Background(a.pal.mutedFill).Children(func() {
			ui.Icon(c, glyph).FontSize(15).TextColor(a.pal.text)
		})
		ui.Text(c, text).FontSize(12.5).LineHeight(1.6).TextColor(a.pal.muted).Grow(1).MinWidth(0)
	})
}

// aboutCard shows the version and the license.
func (a *App) aboutCard(c *ui.Context) {
	a.settingsCard(c, func() {
		ui.Row(c).Gap(14).Padding(14, 16).AlignItems(ui.Center).Children(func() {
			a.logoMark(c, 40)
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "Bucky").FontSize(14).FontWeight(700)
					ui.Text(c, a.tf("settings.version", "version", Version)).FontSize(11.5).
						TextColor(a.pal.muted)
				})
				ui.Text(c, a.t("settings.tagline")).FontSize(12.5).TextColor(a.pal.muted)
			})
			ui.Text(c, "Apache License 2.0").FontSize(11.5).TextColor(a.pal.faint).SingleLine()
		})
	})
}
