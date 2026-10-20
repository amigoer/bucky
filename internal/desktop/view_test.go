package desktop

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/egoist/mygo/ui"
)

// newTestApp builds an application without services, for the views that
// draw only what the state holds.
func newTestApp(t *testing.T) *App {
	t.Helper()
	app := &App{
		msg:          loadMessages(langEnglish),
		prefs:        defaultPrefs(),
		buckets:      map[string]*bucketState{},
		page:         pageBrowse,
		sidebarOpen:  true,
		sidebarWidth: sidebarDefault,
	}
	app.pal = newPalette(false, app.prefs.Accent)
	app.browse.view = viewList
	app.browse.sortKey = "name"
	app.browse.sortDir = 1
	return app
}

// snapshot renders the window once and writes it, so the layout can be
// looked at without starting the application.
func snapshot(t *testing.T, name string, app *App, dark bool) {
	t.Helper()
	if dark {
		app.pal = newPalette(true, app.prefs.Accent)
	}
	tt := ui.NewTester(app.view, 1280, 840)
	if dark {
		tt.SetDark(true)
	}
	tt.Frame()
	writePNG(t, name, tt.Image())
}

func writePNG(t *testing.T, name string, img image.Image) {
	t.Helper()
	if img == nil {
		t.Fatalf("no frame for %s", name)
	}
	dir := os.Getenv("BUCKY_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set BUCKY_SNAPSHOT_DIR to write view snapshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

// withProfile gives the app a connection whose buckets are already read,
// so the views need no service.
func withProfile(app *App) {
	app.profiles = []core.Profile{sampleProfile()}
	app.buckets["p1"] = &bucketState{
		loaded:  true,
		buckets: []core.Bucket{{Name: "assets"}, {Name: "backups"}, {Name: "media"}},
	}
}

func sampleProfile() core.Profile {
	return core.Profile{
		ID: "p1", Name: "Acme assets", ProviderType: "r2",
		AccountID: "0123456789abcdef0123456789abcdef", Bucket: "assets",
		PublicURL: "https://cdn.example.com", CredentialSource: "keychain",
	}
}

func sampleEntries() []entry {
	now := time.Now()
	return []entry{
		{folder: true, key: "images/", name: "images"},
		{folder: true, key: "documents/", name: "documents"},
		{key: "hero-aurora.jpg", name: "hero-aurora.jpg", size: 2_400_000, modified: now, contentType: "image/jpeg", etag: "a1"},
		{key: "notes.md", name: "notes.md", size: 12_800, modified: now, contentType: "text/markdown", etag: "b2"},
		{key: "data.json", name: "data.json", size: 240_000, modified: now, contentType: "application/json", etag: "c3"},
		{key: "archive.zip", name: "archive.zip", size: 91_000_000, modified: now, contentType: "application/zip", etag: "d4"},
		{key: "clip.mp4", name: "clip.mp4", size: 512_000_000, modified: now, contentType: "video/mp4", etag: "e5"},
		{key: "song.mp3", name: "song.mp3", size: 8_400_000, modified: now, contentType: "audio/mpeg", etag: "f6"},
	}
}

func TestWelcomeView(t *testing.T) {
	app := newTestApp(t)
	snapshot(t, "01-welcome", app, false)
}

func TestWelcomeViewDark(t *testing.T) {
	app := newTestApp(t)
	snapshot(t, "02-welcome-dark", app, true)
}

func TestBrowserView(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	app.browse.rows = sampleEntries()
	app.browse.shown = app.browse.rows
	snapshot(t, "03-browser-list", app, false)
}

func TestBrowserViewGrid(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	app.browse.rows = sampleEntries()
	app.browse.shown = app.browse.rows
	app.browse.view = viewGrid
	snapshot(t, "04-browser-grid", app, false)
}

func TestBrowserViewColumns(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets", Prefix: "images/2026/"}
	app.browse.rows = sampleEntries()
	app.browse.shown = app.browse.rows
	app.browse.view = viewColumns
	app.browse.columnRows = map[string][]entry{
		"":             {sampleEntries()[0], sampleEntries()[1]},
		"images/":      {{folder: true, key: "images/2026/", name: "2026"}},
		"images/2026/": sampleEntries()[2:],
	}
	snapshot(t, "05-browser-columns", app, false)
}

func TestBrowserReadOnly(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.profiles[0].ReadOnly = true
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	app.browse.rows = sampleEntries()
	app.browse.shown = app.browse.rows
	app.browse.selection.Add("notes.md")
	app.browse.selection.Add("data.json")
	snapshot(t, "06-browser-selection", app, true)
}

func TestTransfersView(t *testing.T) {
	app := newTestApp(t)
	app.page = pageTransfers
	app.transfers.loaded = true
	now := time.Now()
	app.transfers.tasks = []core.TransferTask{
		{ID: "t1", Direction: "upload", Object: core.ObjectRef{Bucket: "assets", Key: "images/hero-aurora.jpg"},
			Status: "running", Size: 2_400_000, Completed: 1_100_000, BytesPerSecond: 900_000, CreatedAt: now},
		{ID: "t2", Direction: "download", Object: core.ObjectRef{Bucket: "assets", Key: "clip.mp4"},
			Status: "queued", Size: 512_000_000, CreatedAt: now},
		{ID: "t3", Direction: "upload", Object: core.ObjectRef{Bucket: "assets", Key: "archive.zip"},
			Status: "failed", Size: 91_000_000, Completed: 40_000_000, CreatedAt: now},
		{ID: "t4", Direction: "download", Object: core.ObjectRef{Bucket: "assets", Key: "song.mp3"},
			Status: "completed", Size: 8_400_000, Completed: 8_400_000, CreatedAt: now},
	}
	snapshot(t, "07-transfers", app, false)
}

func TestUsageView(t *testing.T) {
	app := newTestApp(t)
	app.page = pageUsage
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	app.usage.location = app.location
	app.usage.usage = core.Usage{
		ProfileID: "p1", Bucket: "assets", Objects: 12_480, Bytes: 91_400_000_000,
		Types:     map[string]int64{"image": 8_200, "video": 320, "text": 2_100, "archive": 60, "other": 1_800},
		Counts:    map[string]int64{".jpg": 5_400, ".png": 2_800, ".md": 2_100, ".mp4": 320, ".zip": 60, "(none)": 1_800},
		ScannedAt: time.Now(), Complete: true,
	}
	snapshot(t, "08-usage", app, false)
}

func TestSettingsView(t *testing.T) {
	app := newTestApp(t)
	app.page = pageSettings
	withProfile(app)
	app.settings.loaded = true
	snapshot(t, "09-settings-general", app, false)
}

func TestSettingsAppearance(t *testing.T) {
	app := newTestApp(t)
	app.page = pageSettings
	app.settings.section = 1
	snapshot(t, "10-settings-appearance", app, true)
}

func TestSettingsIntegrations(t *testing.T) {
	app := newTestApp(t)
	app.page = pageSettings
	app.settings.loaded = true
	app.settings.section = 3
	app.prefs.Language = langChinese
	app.msg = loadMessages(langChinese)
	snapshot(t, "11-settings-integrations", app, false)
}

func TestSettingsAbout(t *testing.T) {
	app := newTestApp(t)
	app.page = pageSettings
	app.settings.section = 5
	snapshot(t, "21-settings-about", app, false)
}

func TestConnectView(t *testing.T) {
	app := newTestApp(t)
	app.openConnect("")
	snapshot(t, "12-connect", app, false)
}

func TestConnectCloudflare(t *testing.T) {
	app := newTestApp(t)
	app.connect = connectState{
		open: true, profile: core.Profile{CredentialSource: "cloudflare"},
		name: "Acme assets", accountID2: "acc1", bucket2: "assets",
		accounts: []service.CloudflareAccount{{ID: "acc1", Name: "Acme"}},
		buckets:  []core.Bucket{{Name: "assets"}, {Name: "backups"}},
	}
	snapshot(t, "13-connect-cloudflare", app, false)
}

func TestPlanView(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	rows := sampleEntries()
	app.browse.rows = rows
	app.browse.shown = rows
	app.plan = planState{
		open: true, operation: "delete",
		location: *app.location, entries: []entry{rows[2], rows[3]},
		ready: true, confirmed: true,
		plan: core.OperationPlan{
			ID: "plan1", Operation: "delete", Bytes: 2_412_800,
			Items: []core.PlanItem{
				{Source: core.ObjectInfo{ObjectRef: core.ObjectRef{Key: "hero-aurora.jpg"}, Size: 2_400_000, ContentType: "image/jpeg"}},
				{Source: core.ObjectInfo{ObjectRef: core.ObjectRef{Key: "notes.md"}, Size: 12_800, ContentType: "text/markdown"}},
			},
			ExpiresAt: time.Now().Add(14 * time.Minute),
		},
	}
	snapshot(t, "14-plan-delete", app, false)
}

func TestShareView(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	rows := sampleEntries()
	app.share = shareState{
		open: true, location: *app.location, entries: []entry{rows[2]},
		mode: "temporary", seconds: 3600, ready: true,
		links: []shareLink{{
			entry: rows[2], url: "https://assets.r2.example.com/hero-aurora.jpg?X-Amz-Expires=3600&X-Amz-Signature=abc",
			expires: "14:32",
		}},
	}
	snapshot(t, "15-share", app, false)
}

func TestQuickView(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.prefs.Quick.ProfileID = "p1"
	app.prefs.Quick.Bucket = "assets"
	app.quick = quickState{open: true, name: "shot.png", size: 420_000}
	snapshot(t, "16-quick", app, false)
}

func TestSearchPalette(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	rows := sampleEntries()
	app.browse.rows = rows
	app.browse.shown = rows
	app.search = searchPaletteState{open: true, query: "im"}
	snapshot(t, "17-search", app, false)
}

func TestConfirmView(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	rows := sampleEntries()
	app.browse.rows = rows
	app.browse.shown = rows
	app.confirm = confirmState{
		open: true, title: "Delete “assets”?", message: "You cannot undo this.",
		confirm: "Delete", destructive: true,
	}
	snapshot(t, "18-confirm", app, false)
}

func TestPreviewView(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.location = &Location{ProfileID: "p1", Bucket: "assets"}
	rows := sampleEntries()
	app.browse.rows = rows
	app.browse.shown = rows
	app.preview = previewState{
		open: true, key: "notes.md", list: rows,
		info: core.ObjectInfo{
			ObjectRef: core.ObjectRef{ProfileID: "p1", Bucket: "assets", Key: "notes.md"},
			Size:      12_800, ETag: "\"9b2cf535f27731c974343645a3985328\"", LastModified: time.Now(),
			ContentType: "text/markdown", Metadata: map[string]string{"x-amz-meta-author": "amigoer"},
		},
		text: core.TextPreview{Text: "# Release notes\n\n- Native desktop shell\n- Plans before every delete\n", ContentType: "text/markdown"},
	}
	snapshot(t, "19-preview-text", app, false)
}

func TestConnectSignIn(t *testing.T) {
	app := newTestApp(t)
	app.openConnect("")
	app.setAuthMethod(true)
	snapshot(t, "20-connect-signin", app, false)
}
