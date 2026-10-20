package desktop

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Version of the desktop application, as the About section shows it.
const Version = "0.1.0"

// Settings sections, in the order the page lists them.
var settingsSections = []string{
	"general", "appearance", "transfers", "integrations", "shortcuts", "about",
}

// mcpSnippet is the client configuration the Integrations section shows.
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

// settingsPage draws the settings with their section list.
func (a *App) settingsPage(c *ui.Context) {
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		a.settingsNav(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).MaxWidth(720).Gap(24).Padding(28, 32, 48, 32).Margin(0, ui.Auto).Children(func() {
				section := settingsSections[a.settings.section]
				ui.Text(c, a.t("settings.sections."+section)).FontSize(20).Bold()
				ui.Text(c, a.t("settings.descriptions."+section)).FontSize(13).
					TextColor(a.pal.muted)
				a.settingsBody(c, section)
			})
		})
	})
}

// settingsNav is the list of sections.
func (a *App) settingsNav(c *ui.Context) {
	glyphs := []*ui.SVG{iconSettings, iconPalette, iconArrowDownUp, iconPlug, iconKeyboard, iconInfo}
	ui.Column(c).Width(200).Padding(12).Gap(2).
		BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border).Children(func() {
		for i, name := range settingsSections {
			active := i == a.settings.section
			row := ui.Row(c).Height(32).PaddingX(10).Gap(10).Radius(6).AlignItems(ui.Center)
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
				ui.Icon(c, glyphs[i]).FontSize(15).TextColor(color)
				text := ui.Text(c, a.t("settings.sections."+name)).FontSize(13).SingleLine()
				if active {
					text.Bold().TextColor(a.pal.primaryText)
				}
			})
			if row.Clicked() {
				a.settings.section = i
			}
		}
	})
}

// settingsBody draws the chosen section.
func (a *App) settingsBody(c *ui.Context, section string) {
	switch section {
	case "general":
		a.settingsGeneral(c)
	case "appearance":
		a.settingsAppearance(c)
	case "transfers":
		a.settingsTransfers(c)
	case "integrations":
		a.settingsIntegrations(c)
	case "shortcuts":
		a.settingsShortcuts(c)
	case "about":
		a.settingsAbout(c)
	}
}

// settingsGroup is a titled card of settings rows.
func (a *App) settingsGroup(c *ui.Context, title string, rows func()) {
	ui.Column(c).Gap(2).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(11.5).Bold().TextColor(a.pal.faint).Padding(0, 4, 4, 4)
		}
		card := ui.Column(c).Gap(2).Padding(4).Radius(12).
			Background(a.pal.card).Border(1, a.pal.border)
		card.Children(rows)
	})
}

// settingsRow is a label, its explanation and the control.
func (a *App) settingsRow(c *ui.Context, title, description string, control func()) {
	ui.Row(c).Padding(12, 14).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(13).Bold().SingleLine()
			if description != "" {
				ui.Text(c, description).FontSize(12).TextColor(a.pal.muted)
			}
		})
		control()
	})
}

// settingsGeneral is the language, the folder view and the usage entry.
func (a *App) settingsGeneral(c *ui.Context) {
	a.settingsGroup(c, "", func() {
		a.settingsRow(c, a.t("settings.language"), a.t("settings.languageDesc"), func() {
			index := 0
			if a.prefs.Language == langChinese {
				index = 1
			}
			seg := ui.Segmented(c, &index, "English", "简体中文").Label(a.t("settings.language"))
			if seg.Changed() {
				if index == 1 {
					a.setLanguage(langChinese)
				} else {
					a.setLanguage(langEnglish)
				}
			}
		})
		a.settingsRow(c, a.t("settings.view"), a.tf("settings.viewDesc", "keys", "⌘1–3"),
			func() {
				views := []string{viewList, viewGrid, viewColumns}
				index := 0
				for i, name := range views {
					if name == a.prefs.View {
						index = i
					}
				}
				labels := []string{
					a.t("settings.viewList"), a.t("settings.viewIcons"), a.t("settings.viewColumns"),
				}
				seg := ui.Segmented(c, &index, labels...).Label(a.t("settings.view"))
				if seg.Changed() && index >= 0 && index < len(views) {
					a.browse.view = views[index]
					a.prefs.View = views[index]
					a.savePrefs()
				}
			})
	})
	a.settingsGroup(c, "", func() {
		a.settingsRow(c, a.t("settings.usage"), a.t("settings.usageHint"), func() {
			if ui.Button(c, a.t("settings.openUsage")).Clicked() {
				a.page = pageUsage
				a.ensureUsage()
			}
		})
	})
}

// settingsAppearance is the theme and the accent color.
func (a *App) settingsAppearance(c *ui.Context) {
	a.settingsGroup(c, "", func() {
		a.settingsRow(c, a.t("settings.theme"), a.t("settings.themeDesc"), func() {
			themes := []string{themeLight, themeDark, themeSystem}
			index := 0
			for i, name := range themes {
				if name == a.prefs.Theme {
					index = i
				}
			}
			labels := []string{a.t("settings.light"), a.t("settings.dark"), a.t("settings.system")}
			seg := ui.Segmented(c, &index, labels...).Label(a.t("settings.theme"))
			if seg.Changed() && index >= 0 && index < len(themes) {
				a.prefs.Theme = themes[index]
				a.savePrefs()
				a.refreshPalette()
			}
		})
		a.settingsRow(c, a.t("settings.accent"), a.t("settings.accentDesc"), func() {
			ui.Row(c).Gap(8).Children(func() {
				for _, name := range accentNames {
					a.accentSwatch(c, name)
				}
			})
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
		button.Border(2, a.pal.text)
	} else {
		button.Border(1, a.pal.borderStrong)
	}
	if button.Clicked() {
		a.prefs.Accent = name
		a.savePrefs()
		a.refreshPalette()
	}
}

// settingsTransfers is the speed limit and the concurrency.
func (a *App) settingsTransfers(c *ui.Context) {
	a.settingsGroup(c, "", func() {
		a.settingsRow(c, a.t("settings.rateLimit"), a.t("settings.rateLimitHint"), func() {
			options := make([]string, 0, len(rates))
			for _, rate := range rates {
				if rate == 0 {
					options = append(options, a.t("transfers.unlimited"))
					continue
				}
				options = append(options, trimZero(rate)+" "+a.t("transfers.perSecond"))
			}
			current := options[0]
			for i, rate := range rates {
				if rate == a.prefs.Rate {
					current = options[i]
				}
			}
			select_ := ui.Select(c, &current, options).Label(a.t("settings.rateLimit")).Width(144)
			if !select_.Changed() {
				return
			}
			for i, option := range options {
				if option == current {
					a.prefs.Rate = rates[i]
					a.savePrefs()
					go func(bytes int64) { a.svc.Transfers.SetRate(bytes) }(int64(rates[i] * 1024 * 1024))
					return
				}
			}
		})
		a.settingsRow(c, a.t("settings.concurrency"), a.t("settings.concurrencyHint"), func() {
			options := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
			current := strconv.Itoa(a.prefs.Concurrency)
			select_ := ui.Select(c, &current, options).Label(a.t("settings.concurrency")).Width(144)
			if !select_.Changed() {
				return
			}
			value, err := strconv.Atoi(current)
			if err != nil || value < 1 {
				return
			}
			a.prefs.Concurrency = value
			a.savePrefs()
			go func() { a.svc.Transfers.SetConcurrency(value) }()
		})
	})
	a.settingsGroup(c, "", func() {
		a.settingsRow(c, a.t("settings.transferQueue"), a.t("settings.transferNote"), func() {
			if ui.Button(c, a.t("settings.openTransfers")).Clicked() {
				a.page = pageTransfers
				a.loadTasks()
			}
		})
	})
}

// settingsIntegrations holds the Cloudflare sign-in settings and the agent
// entry points.
func (a *App) settingsIntegrations(c *ui.Context) {
	a.loadSettings()
	a.cloudflareCard(c)
	a.integrationCard(c, iconPlug, "MCP", a.t("settings.mcpDesc"), mcpSnippet, func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Text(c, a.t("settings.tools")).FontSize(12).TextColor(a.pal.muted)
			for _, tool := range mcpTools {
				ui.Badge(c, tool).Background(a.pal.field).TextColor(a.pal.muted)
			}
		})
	})
	a.integrationCard(c, iconSquareTerminal, "CLI", a.t("settings.cliDesc"), cliSnippet, nil)
	a.integrationCard(c, iconBot, "Agent Skill", a.t("settings.skillDesc"), skillSnippet, nil)
}

// integrationCard is one agent entry point with the snippet to copy.
func (a *App) integrationCard(c *ui.Context, glyph *ui.SVG, title, description, code string, extra func()) {
	card := ui.Column(c).Gap(12).Padding(16).Radius(12).
		Background(a.pal.card).Border(1, a.pal.border)
	card.Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
			ui.Box(c).Size(32, 32).Radius(8).Center().Background(a.pal.field).Children(func() {
				ui.Icon(c, glyph).FontSize(16).TextColor(a.pal.muted)
			})
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, title).FontSize(13).Bold()
				ui.Text(c, description).FontSize(12).TextColor(a.pal.muted)
			})
		})
		a.codeBlock(c, code)
		if extra != nil {
			extra()
		}
	})
}

// codeBlock shows a snippet with a button that copies it.
func (a *App) codeBlock(c *ui.Context, code string) {
	block := ui.Column(c).Gap(6).Padding(10, 12).Radius(8).Background(a.pal.field)
	block.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
			ui.Text(c, code).FontSize(11.5).Font("monospace").Grow(1).Selectable()
			copy := ui.Button(c, "").Label(a.t("common.copy")).Tooltip(a.t("common.copy")).Size(24, 24)
			copy.Children(func() { ui.Icon(c, iconCopy).FontSize(13) })
			if copy.Clicked() {
				a.copyText(c, code)
			}
		})
	})
}

// cloudflareCard holds the OAuth client settings the sign-in needs.
func (a *App) cloudflareCard(c *ui.Context) {
	card := ui.Column(c).Gap(12).Padding(16).Radius(12).
		Background(a.pal.card).Border(1, a.pal.border)
	card.Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
			ui.Box(c).Size(32, 32).Radius(8).Center().Background(a.pal.field).Children(func() {
				ui.Icon(c, iconGlobe).FontSize(16).TextColor(a.pal.muted)
			})
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, a.t("settings.cloudflare")).FontSize(13).Bold()
				ui.Text(c, a.t("settings.cloudflareDesc")).FontSize(12).TextColor(a.pal.muted)
			})
		})
		ui.Text(c, a.t("settings.cloudflareSetup")).FontSize(11.5).TextColor(a.pal.muted)
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, a.t("settings.cloudflareClientId")).FontSize(11.5).TextColor(a.pal.faint)
			input := ui.TextInput(c, &a.settings.clientID).
				Placeholder(a.t("settings.cloudflareClientId")).Width(360)
			if input.Submitted() {
				a.saveCloudflareSettings()
			}
		})
		ui.Text(c, a.t("settings.cloudflareClientIdHint")).FontSize(11).TextColor(a.pal.faint)
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, a.t("settings.cloudflareRedirect")).FontSize(11.5).TextColor(a.pal.faint)
			input := ui.TextInput(c, &a.settings.redirect).Width(360)
			if input.Submitted() {
				a.saveCloudflareSettings()
			}
		})
		ui.Text(c, a.t("settings.cloudflareRedirectHint")).FontSize(11).TextColor(a.pal.faint)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, a.t("common.save")).Clicked() {
				a.saveCloudflareSettings()
			}
			if ui.Button(c, a.t("settings.cloudflareDocs")).Clicked() {
				_ = mygo.Shell.OpenExternal("https://developers.cloudflare.com/fundamentals/api/get-started/create-token/")
			}
			if a.settings.notice != "" {
				ui.Text(c, a.settings.notice).FontSize(11.5).TextColor(a.pal.successText)
			}
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

// settingsShortcuts lists the keyboard shortcuts.
func (a *App) settingsShortcuts(c *ui.Context) {
	mod := "Ctrl"
	if platformIsMac() {
		mod = "⌘"
	}
	groups := []struct {
		title string
		items []struct{ keys, label string }
	}{
		{a.t("settings.shortcutGroups.browse"), []struct{ keys, label string }{
			{mod + "F", a.t("keys.search")},
			{mod + "1–3", a.t("keys.views")},
			{"↵", a.t("keys.open")},
			{mod + "[ / ]", a.t("keys.history")},
		}},
		{a.t("settings.shortcutGroups.files"), []struct{ keys, label string }{
			{mod + "A", a.t("keys.selectAll")},
			{"Space", a.t("keys.quickLook")},
			{a.t("keys.delete"), mod + "⌫"},
			{"Esc", a.t("keys.close")},
		}},
		{a.t("settings.shortcutGroups.app"), []struct{ keys, label string }{
			{a.t("keys.quickUpload"), mod + "⌥U"},
			{a.t("keys.transfers"), mod + "⇧T"},
			{a.t("keys.settings"), mod + ","},
			{a.t("keys.sidebar"), mod + "⌃S"},
		}},
	}
	for _, group := range groups {
		a.settingsGroup(c, group.title, func() {
			for _, item := range group.items {
				ui.Row(c).Height(40).PaddingX(14).Gap(16).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.label).FontSize(13).SingleLine().Grow(1)
					ui.Badge(c, item.keys).Background(a.pal.field).TextColor(a.pal.muted)
				})
			}
		})
	}
}

// settingsAbout shows the version and the promises the application makes.
func (a *App) settingsAbout(c *ui.Context) {
	card := ui.Column(c).Gap(6).Padding(28, 24).Radius(12).AlignItems(ui.Center).
		Background(a.pal.card).Border(1, a.pal.border)
	card.Children(func() {
		if mark := appLogo(); mark != nil {
			ui.Image(c, mark).Size(64, 64).Radius(16).Clip()
		}
		ui.Text(c, "Bucky").FontSize(20).Bold()
		ui.Text(c, a.t("settings.tagline")).FontSize(13).TextColor(a.pal.muted)
		ui.Badge(c, a.tf("settings.version", "version", Version)).
			Background(a.pal.field).TextColor(a.pal.muted)
	})
	a.settingsGroup(c, "", func() {
		a.aboutRow(c, iconLock, a.t("settings.privacy"), a.t("settings.privacyDesc"))
		a.aboutRow(c, iconShieldCheck, a.t("settings.safety"), a.t("settings.safetyDesc"))
		a.aboutRow(c, iconScale, a.t("settings.license"), "Apache License 2.0")
	})
}

func (a *App) aboutRow(c *ui.Context, glyph *ui.SVG, title, description string) {
	ui.Row(c).Padding(12, 14).Gap(12).AlignItems(ui.Start).Children(func() {
		ui.Icon(c, glyph).FontSize(16).TextColor(a.pal.muted)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(13).Bold()
			ui.Text(c, description).FontSize(12).TextColor(a.pal.muted)
		})
	})
}
