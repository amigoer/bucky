// Package desktop is the Bucky desktop application. It draws the interface
// with MyGo's native UI toolkit and calls the application services in
// process, so no webview, no page and no bridge take part.
package desktop

import (
	"context"
	"errors"
	"log"
	"math"
	"runtime"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/amigoer/bucky/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Pages the window can show.
const (
	pageBrowse    = "browse"
	pageTransfers = "transfers"
	pageUsage     = "usage"
	pageSettings  = "settings"
)

// Title bar metrics. The window controls the system draws sit in the same
// row, which Context.TitleBar reports the room of.
const (
	titleBarHeight = 52.0
	sidebarDefault = 240.0
	sidebarMin     = 200.0
	sidebarMax     = 360.0
	// Left edge of the window controls on macOS, as the design places them.
	trafficLightLeft = 18.0
	// Diameter of a macOS window control. TrafficLightPosition takes the
	// top-left corner, so the controls start half a control above the
	// centre of the header row.
	trafficLightSize = 14.0
	// Width of the three macOS window controls and the gaps between them.
	trafficLightsWidth = 52.0
)

// Location is the folder the browser shows.
type Location struct {
	ProfileID string
	Bucket    string
	Prefix    string
}

func sameLocation(a, b *Location) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.ProfileID == b.ProfileID && a.Bucket == b.Bucket && a.Prefix == b.Prefix
}

// App is the desktop application: its state, its windows and the services
// it calls.
type App struct {
	svc   *service.Service
	store *prefsStore
	prefs *prefs
	msg   *messages
	// pal is the palette the frame draws with. It fades toward palTarget,
	// which follows the appearance and accent.
	pal       *palette
	palTarget *palette
	win       *mygo.Window

	ctx context.Context

	// Routing.
	page         string
	location     *Location
	backStack    []Location
	forwardStack []Location

	// Connections and their buckets.
	profiles     []core.Profile
	profilesErr  error
	profilesBusy bool
	buckets      map[string]*bucketState

	// Interface state that does not outlive the run.
	sidebarOpen  bool
	sidebarWidth float32
	appearance   int // bumped when the palette is rebuilt

	// pendingToast is a message raised from a goroutine, shown by the next
	// frame.
	pendingToast string

	browse    browseState
	transfers transfersState
	usage     usageState
	settings  settingsState

	preview previewState
	plan    planState
	share   shareState
	connect connectState
	quick   quickState
	confirm confirmState
	search  searchPaletteState
}

// New builds the application over the services.
func New(ctx context.Context, backend *service.Service) (*App, error) {
	dir, err := store.DataDir()
	if err != nil {
		return nil, err
	}
	store := newPrefsStore(dir)
	prefs := store.load()
	app := &App{
		svc:          backend,
		ctx:          ctx,
		store:        store,
		prefs:        prefs,
		msg:          loadMessages(prefs.Language),
		buckets:      map[string]*bucketState{},
		page:         pageBrowse,
		sidebarOpen:  !prefs.SidebarHidden,
		sidebarWidth: sidebarDefault,
	}
	app.browse.view = prefs.View
	app.browse.sortKey = prefs.SortKey
	app.browse.sortDir = prefs.SortDir
	if app.browse.view == "" {
		app.browse.view = viewList
	}
	if app.browse.sortDir == 0 {
		app.browse.sortDir = 1
	}
	// The desktop's appearance is only readable once the app is running, so
	// the palette starts light and is rebuilt when the window is made.
	app.palTarget = newPalette(false, prefs.Accent)
	app.pal = app.palTarget
	return app, nil
}

// Run shows the window and runs the application until it quits.
func (a *App) Run() error {
	mygo.App.WhenReady(func() {
		a.refreshPalette()
		options := mygo.WindowOptions{
			Title:         "Bucky",
			Width:         1280,
			Height:        840,
			MinWidth:      960,
			MinHeight:     640,
			StateKey:      "main",
			TitleBarStyle: mygo.TitleBarHidden,
			Content:       ui.View(a.view),
		}
		if platformIsMac() {
			// Centre the traffic lights on the header row, as the design
			// places them.
			options.TrafficLightPosition = &mygo.Point{
				X: int(trafficLightLeft),
				Y: int(titleBarHeight/2 - trafficLightSize/2),
			}
		} else {
			// The system's own window buttons fill the bar at the top right,
			// so give them the header row's height to sit in.
			options.TitleBarHeight = int(titleBarHeight)
		}
		a.win = mygo.NewWindow(options)
		a.sweepStaged(a.ctx)
		a.loadProfiles()
		a.loadTasks()
		a.watchTransfers()
	})
	mygo.Theme.OnUpdated(func() { a.applyAppearance() })
	return mygo.App.Run()
}

// Close releases the services when the window goes.
func (a *App) Close() {
	if a.svc != nil {
		_ = a.svc.Close()
	}
}

// applyAppearance rebuilds the palette when the desktop's appearance changes.
func (a *App) applyAppearance() {
	a.refreshPalette()
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// darkMode reports whether the interface draws dark.
func (a *App) darkMode() bool {
	switch a.prefs.Theme {
	case themeLight:
		return false
	case themeDark:
		return true
	}
	return mygo.Theme.IsDark()
}

func (a *App) refreshPalette() {
	a.palTarget = newPalette(a.darkMode(), a.prefs.Accent)
	a.appearance++
}

// t returns a string of the current language.
func (a *App) t(key string) string { return a.msg.t(key) }

// tf returns a string with {name} placeholders replaced.
func (a *App) tf(key string, args ...any) string { return a.msg.tf(key, args...) }

// tn picks the singular or plural form of a counted string.
func (a *App) tn(key string, count int, args ...any) string {
	return a.msg.tn(key, count, args...)
}

// errorText is the user-facing text of a service error.
func (a *App) errorText(err error) string {
	if err == nil {
		return ""
	}
	info := core.AsError(err)
	if info.Code == "invalid_argument" && info.Message != "" {
		return info.Message
	}
	if knownErrorCodes[info.Code] {
		return a.t("errors." + info.Code)
	}
	if info.Code == "internal" || info.Code == "" {
		return a.t("errors.internal")
	}
	if info.Message != "" {
		return info.Message
	}
	return a.t("errors.internal")
}

var knownErrorCodes = map[string]bool{
	"read_only": true, "conflict": true, "expired": true, "confirmation_required": true,
	"not_found": true, "forbidden": true, "configuration": true, "protocol": true,
	"local_io": true, "dialog": true, "clipboard": true, "cancelled": true,
	"timeout": true, "credentials_unavailable": true, "task_busy": true,
	"unsupported": true, "closed": true, "storage": true, "cleanup_failed": true,
	"network": true, "desktop_required": true, "permission_denied": true,
	"rate_limited": true, "unavailable": true, "provider_error": true, "browser": true,
}

// isCredentialError reports whether the connection needs its credentials
// looked at again.
func isCredentialError(err error) bool {
	code := core.AsError(err).Code
	return code == "credentials_unavailable" || code == "forbidden" || code == "configuration"
}

// update runs fn on the UI thread and draws a new frame.
func (a *App) update(fn func()) {
	if a.win == nil {
		return
	}
	a.win.Update(fn)
}

// progressBar draws work done in a color of its own, as a progress bar
// that tells a finished transfer from one in trouble.
func (a *App) progressBar(c *ui.Context, ratio float64, height float32, color ui.Color) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	ui.Box(c).Height(height).Radius(height / 2).Clip().Background(a.pal.field).Children(func() {
		ui.Box(c).WidthPercent(float32(ratio * 100)).FillHeight().
			Radius(height / 2).Background(color)
	})
}

// toastCurrent shows a message raised outside a build pass.
func (a *App) toastCurrent(message string) {
	if message == "" {
		return
	}
	a.pendingToast = message
	a.update(func() {})
}

// flushToast shows the message a goroutine raised.
func (a *App) flushToast(c *ui.Context) {
	if a.pendingToast == "" {
		return
	}
	message := a.pendingToast
	a.pendingToast = ""
	c.Toast(message)
}

// copyText puts text on the clipboard, reporting a failure in a toast.
func (a *App) copyText(c *ui.Context, text string) {
	mygo.Clipboard.WriteText(text)
	c.Toast(a.t("toast.copied"))
}

// savePrefs writes the preferences and rebuilds what they change.
func (a *App) savePrefs() {
	if err := a.store.save(a.prefs); err != nil {
		log.Printf("bucky: cannot save preferences: %v", err)
	}
}

// setLanguage switches the interface language.
func (a *App) setLanguage(lang string) {
	if lang == a.prefs.Language {
		return
	}
	a.prefs.Language = lang
	a.msg = loadMessages(lang)
	a.savePrefs()
}

// profileOf finds a connection by its ID.
func (a *App) profileOf(id string) *core.Profile {
	for i := range a.profiles {
		if a.profiles[i].ID == id {
			return &a.profiles[i]
		}
	}
	return nil
}

// currentProfile is the connection the browser is showing.
func (a *App) currentProfile() *core.Profile {
	if a.location == nil {
		return nil
	}
	return a.profileOf(a.location.ProfileID)
}

// readOnly reports whether the connection in view refuses writes.
func (a *App) readOnly() bool {
	profile := a.currentProfile()
	return profile != nil && profile.ReadOnly
}

// navigate moves the browser to a folder, keeping the history.
func (a *App) navigate(location Location) {
	if sameLocation(a.location, &location) {
		a.page = pageBrowse
		return
	}
	if a.location != nil {
		a.backStack = append(trimStack(a.backStack), *a.location)
	}
	a.forwardStack = nil
	a.enter(location)
}

// enter moves the browser to a folder without touching the history.
func (a *App) enter(location Location) {
	a.location = &location
	a.page = pageBrowse
	a.browse.reset(location)
	a.ensureBuckets(location.ProfileID)
	a.loadEntries()
}

func (a *App) goBack() {
	if len(a.backStack) == 0 {
		return
	}
	target := a.backStack[len(a.backStack)-1]
	a.backStack = a.backStack[:len(a.backStack)-1]
	if a.location != nil {
		a.forwardStack = append(trimStack(a.forwardStack), *a.location)
	}
	a.enter(target)
}

func (a *App) goForward() {
	if len(a.forwardStack) == 0 {
		return
	}
	target := a.forwardStack[len(a.forwardStack)-1]
	a.forwardStack = a.forwardStack[:len(a.forwardStack)-1]
	if a.location != nil {
		a.backStack = append(trimStack(a.backStack), *a.location)
	}
	a.enter(target)
}

// trimStack keeps the history bounded.
func trimStack(stack []Location) []Location {
	const limit = 50
	if len(stack) >= limit {
		return stack[len(stack)-limit+1:]
	}
	return stack
}

// forgetProfile drops a connection from the history after it is removed.
func (a *App) forgetProfile(id string) {
	keep := func(l Location) bool { return l.ProfileID != id }
	back := a.backStack[:0]
	for _, l := range a.backStack {
		if keep(l) {
			back = append(back, l)
		}
	}
	a.backStack = back
	forward := a.forwardStack[:0]
	for _, l := range a.forwardStack {
		if keep(l) {
			forward = append(forward, l)
		}
	}
	a.forwardStack = forward
	if a.location != nil && !keep(*a.location) {
		a.location = nil
	}
	delete(a.buckets, id)
}

// page returns the content of the window below its title bar.
func (a *App) content(c *ui.Context) {
	switch a.page {
	case pageBrowse:
		if a.location == nil {
			a.welcome(c)
			return
		}
		a.browser(c)
	case pageTransfers:
		a.transfersPage(c)
	case pageUsage:
		a.usagePage(c)
	case pageSettings:
		a.settingsPage(c)
	}
}

// view builds the window.
func (a *App) view(c *ui.Context) {
	a.pal = a.palTarget.faded(c.Root())
	c.SetTheme(a.pal.widgetTheme())
	c.Root().Background(a.pal.background)

	bar := c.TitleBar()
	// The header row keeps the design's height. TitleBar.Height would be
	// larger, since it counts the room above and below the window controls,
	// which are centred in this row instead.
	top := float32(titleBarHeight)

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		if a.sidebarOpen {
			a.sidebar(c, top, bar)
		}
		ui.Column(c).Grow(1).Background(a.pal.background).Children(func() {
			a.titleRow(c, top, bar)
			ui.Column(c).Grow(1).Children(func() { a.content(c) })
		})
	})

	// Overlays are built after the content, above it.
	a.overlays(c)
	a.flushToast(c)
}

// titleRow draws the header of the page. With the sidebar hidden it also
// clears the window controls and carries the toggle that brings it back.
func (a *App) titleRow(c *ui.Context, height float32, bar ui.TitleBar) {
	lead := float32(18)
	if a.page == pageBrowse && a.location != nil {
		lead = 12
	}
	row := ui.Row(c).Height(height).Padding(0, bar.Right+14, 0, lead).
		AlignItems(ui.Center).Gap(12).DragWindow().
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border)
	row.Children(func() {
		if !a.sidebarOpen {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				if controls := a.windowControlsWidth(bar); controls > 0 {
					ui.Box(c).Width(controls)
				}
				a.sidebarToggle(c)
			})
		}
		switch a.page {
		case pageTransfers:
			a.pageTitle(c, a.t("transfers.title"))
			a.transfersToolbar(c)
		case pageUsage:
			a.pageTitle(c, a.t("usage.title"))
			a.usageToolbar(c)
		case pageSettings:
			a.pageTitle(c, a.t("settings.title"))
			ui.Spacer(c)
		default:
			if a.location != nil {
				a.browserToolbar(c)
			} else {
				ui.Spacer(c)
			}
		}
	})
}

// windowControlsWidth is the room the macOS window controls take from the
// start of a header row that begins at the window edge.
func (a *App) windowControlsWidth(bar ui.TitleBar) float32 {
	if !platformIsMac() || bar.Left <= 0 {
		return 0
	}
	return trafficLightsWidth
}

func (a *App) pageTitle(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(15).FontWeight(650).SingleLine().Shrink(0)
}

// segments is a segmented control of labels with a raised thumb on the
// chosen one. It reports a new choice.
func (a *App) segments(c *ui.Context, index *int, label string, labels []string) bool {
	seg := ui.SegmentedBase(c, index, len(labels))
	seg.Track.Padding(2).Radius(8).Background(a.pal.mutedFill).Label(label).Shrink(0).Children(func() {
		for i, text := range labels {
			segment := seg.Segment(i).Height(24).PaddingX(10).Center().Radius(6)
			color, weight := a.pal.muted, 500
			if i == *index {
				segment.Background(a.pal.thumb).Shadow(0, 1, 2, 0, a.pal.thumbShadow)
				color, weight = a.pal.text, 600
			}
			segment.Children(func() {
				ui.Text(c, text).FontSize(12).FontWeight(weight).TextColor(color).SingleLine()
			})
		}
	})
	return seg.Track.Changed()
}

// primaryAction is a page's one main action, in the accent color.
func (a *App) primaryAction(c *ui.Context, glyph *ui.SVG, label string, disabled bool) ui.Element {
	button := ui.ButtonBase(c).Label(label).Height(28).PaddingX(12).Radius(7).Disabled(disabled).
		Background(a.pal.primary).Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.12))
	if disabled {
		button.Opacity(0.45)
	} else if button.Hovered() {
		button.Background(a.pal.primaryHover)
	}
	button.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if glyph != nil {
				ui.Icon(c, glyph).FontSize(14).TextColor(a.pal.onPrimary)
			}
			ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(a.pal.onPrimary).SingleLine()
		})
	})
	return button
}

// secondaryButton is a quiet button with a face of its own, for actions
// beside the primary one.
func (a *App) secondaryButton(c *ui.Context, glyph *ui.SVG, label string) ui.Element {
	button := ui.ButtonBase(c).Label(label).Height(28).PaddingX(11).Radius(7).
		Background(a.pal.button).Border(0.5, a.pal.buttonRing).Shadow(0, 1, 1, 0, a.pal.buttonShadow)
	button.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if glyph != nil {
				ui.Icon(c, glyph).FontSize(14).TextColor(a.pal.text)
			}
			ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(a.pal.text).SingleLine()
		})
	})
	return button
}

// sidebarToggle shows and hides the sidebar.
func (a *App) sidebarToggle(c *ui.Context) {
	label := a.t("sidebar.hide")
	if !a.sidebarOpen {
		label = a.t("sidebar.show")
	}
	a.ghostButton(c, iconPanelLeft, label, 28, false, a.toggleSidebar)
}

// plainButton is a button with no face of its own until the pointer is
// over it, as the design's toolbar and section controls are. A width of 0
// fits the content.
func (a *App) plainButton(c *ui.Context, label string, width, height float32) ui.Element {
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Height(height).Radius(7)
	if width > 0 {
		button.Width(width)
	}
	if button.Hovered() {
		button.Background(a.pal.hover)
	}
	return button
}

// ghostButton is an icon-only plainButton. An active button keeps a face,
// for the page it is on.
func (a *App) ghostButton(c *ui.Context, glyph *ui.SVG, label string, size float32, active bool, action func()) {
	button := a.plainButton(c, label, size, size)
	if active {
		button.Background(a.pal.hover)
	}
	color := a.pal.muted
	if active || button.Hovered() {
		color = a.pal.text
	}
	button.Children(func() { ui.Icon(c, glyph).FontSize(size * 0.56).TextColor(color) })
	if button.Clicked() {
		action()
	}
}

func (a *App) toggleSidebar() {
	a.sidebarOpen = !a.sidebarOpen
	a.prefs.SidebarHidden = !a.sidebarOpen
	a.savePrefs()
}

// overlays builds the dialogs and panels above the content.
func (a *App) overlays(c *ui.Context) {
	a.previewOverlay(c)
	a.shareOverlay(c)
	a.planOverlay(c)
	a.quickOverlay(c)
	a.connectOverlay(c)
	a.confirmOverlay(c)
	a.searchOverlay(c)
	a.shortcuts(c)
	a.paletteShortcuts(c)
}

// sidebarWidthClamped keeps the sidebar within its limits.
func (a *App) sidebarWidthClamped() float32 {
	width := a.sidebarWidth
	if width <= 0 {
		width = sidebarDefault
	}
	return float32(math.Min(math.Max(float64(width), sidebarMin), sidebarMax))
}

// platformIsMac reports the platform the window controls sit on the left of.
func platformIsMac() bool { return runtime.GOOS == "darwin" }

// errCancelled reports whether an error is the cancellation of a request.
func errCancelled(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || core.AsError(err).Code == "cancelled")
}
