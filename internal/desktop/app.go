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
	panelInset     = 8.0
	panelRadius    = 16.0
	sidebarDefault = 248.0
	sidebarMin     = 200.0
	sidebarMax     = 360.0
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
	pal   *palette
	win   *mygo.Window

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
	app.browse.sort = ui.SortOrder{Column: prefs.SortKey, Descending: prefs.SortDir < 0}
	if app.browse.view == "" {
		app.browse.view = viewList
	}
	if app.browse.sortDir == 0 {
		app.browse.sortDir = 1
	}
	// The desktop's appearance is only readable once the app is running, so
	// the palette starts light and is rebuilt when the window is made.
	app.pal = newPalette(false, prefs.Accent)
	return app, nil
}

// Run shows the window and runs the application until it quits.
func (a *App) Run() error {
	mygo.App.WhenReady(func() {
		a.refreshPalette()
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:         "Bucky",
			Width:         1280,
			Height:        840,
			MinWidth:      960,
			MinHeight:     640,
			StateKey:      "main",
			TitleBarStyle: mygo.TitleBarHidden,
			Content:       ui.View(a.view),
		})
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
	a.pal = newPalette(a.darkMode(), a.prefs.Accent)
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
	c.SetTheme(a.pal.widgetTheme())
	root := c.Root().Background(a.pal.sidebar)

	bar := c.TitleBar()
	top := float32(titleBarHeight)
	if bar.Height > top {
		top = bar.Height
	}

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		if a.sidebarOpen {
			a.sidebar(c, top, bar)
		}
		panel := ui.Column(c).Grow(1).Margin(panelInset).Radius(panelRadius).
			Background(a.pal.background).Border(1, a.pal.border)
		panel.Children(func() {
			ui.Column(c).Fill().Radius(panelRadius).Clip().Children(func() {
				a.titleRow(c, top, bar)
				ui.Column(c).Grow(1).Children(func() { a.content(c) })
			})
		})
	})

	// Overlays are built after the content, above it.
	a.overlays(c)
	a.flushToast(c)
	_ = root
}

// titleRow draws the row the window controls share with the page heading.
func (a *App) titleRow(c *ui.Context, height float32, bar ui.TitleBar) {
	ui.Row(c).Height(height).Padding(0, bar.Right+14, 0, bar.Left+14).
		AlignItems(ui.Center).Gap(8).DragWindow().Children(func() {
		if !a.sidebarOpen {
			a.sidebarToggle(c)
		}
		switch a.page {
		case pageTransfers:
			a.pageTitle(c, a.t("transfers.title"))
		case pageUsage:
			a.pageTitle(c, a.t("usage.title"))
		case pageSettings:
			a.pageTitle(c, a.t("settings.title"))
		default:
			if a.location != nil {
				a.browserToolbar(c)
			} else {
				ui.Spacer(c)
			}
		}
	})
}

func (a *App) pageTitle(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(15).Bold()
	ui.Spacer(c)
}

// sidebarToggle shows and hides the sidebar.
func (a *App) sidebarToggle(c *ui.Context) {
	button := ui.Button(c, "").Label(a.t("sidebar.hide")).Tooltip(a.t("sidebar.hide")).Size(28, 28)
	button.Children(func() { ui.Icon(c, iconPanelLeft).FontSize(16) })
	if button.Clicked() {
		a.toggleSidebar()
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
