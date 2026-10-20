package desktop

import (
	"context"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/egoist/mygo/ui"
)

// bucketState is the bucket list of one connection.
type bucketState struct {
	buckets []core.Bucket
	err     error
	busy    bool
	loaded  bool
}

// searchState is the search running in the folder being browsed.
type searchState struct {
	active   bool
	query    string
	objects  []core.ObjectInfo
	scanned  int64
	complete bool
	cursor   string
	busy     bool
	more     bool
	err      error
}

// browseState is everything the file browser shows.
type browseState struct {
	pages    []core.ListPage
	rows     []entry
	shown    []entry
	loading  bool
	more     bool
	moreBusy bool
	err      error
	cursor   string

	search searchState

	selection ui.Selection[string]
	list      ui.ListState
	grid      ui.GridState
	sort      ui.SortOrder

	view    string
	sortKey string
	sortDir int

	focus string
	// selectedIndex is the row the keyboard moves from.
	selectedIndex int

	// The columns view keeps one column per prefix of the current path.
	columnRows  map[string][]entry
	columnBusy  map[string]bool
	columnErr   map[string]error
	columnLists map[string]*ui.ListState
}

func (b *browseState) reset(location Location) {
	b.pages = nil
	b.rows = nil
	b.shown = nil
	b.err = nil
	b.cursor = ""
	b.more = false
	b.moreBusy = false
	b.search = searchState{}
	b.selection.Clear()
	b.focus = ""
	b.selectedIndex = -1
	b.columnRows = map[string][]entry{}
	b.columnBusy = map[string]bool{}
	b.columnErr = map[string]error{}
	b.columnLists = map[string]*ui.ListState{}
}

// planRequest asks for an operation plan: either entries to plan for, or a
// plan the service already made.
type planRequest struct {
	operation string
	location  Location
	entries   []entry
	plan      *core.OperationPlan
}

// planState is the plan the user is reviewing before it runs.
type planState struct {
	open      bool
	operation string
	location  Location
	entries   []entry

	// target and base are the destination of a copy or a move.
	target core.ObjectRef
	base   string

	plan      core.OperationPlan
	ready     bool
	creating  bool
	running   bool
	progress  int
	total     int
	err       error
	confirmed bool
	queue     []planRequest

	pickerOpen bool
	picker     pickerState
}

// pickerState is the folder chooser a copy or move plan uses.
type pickerState struct {
	location Location
	rows     []entry
	loading  bool
	err      error
	path     []crumb
}

// shareState is the share dialog.
type shareState struct {
	open     bool
	location Location
	entries  []entry

	links   []shareLink
	busy    bool
	err     error
	mode    string
	seconds int
	ready   bool
}

// shareLink is one object and the link made for it.
type shareLink struct {
	entry   entry
	url     string
	headers map[string]string
	expires string
	err     error
}

// connectState is the add-or-edit connection page.
type connectState struct {
	open    bool
	editing string
	profile core.Profile

	name             string
	accountID        string
	bucket           string
	jurisdiction     string
	publicURL        string
	readOnly         bool
	credentialSource string
	accessKeyID      string
	secretKey        string

	keychainSaved bool
	busy          bool
	testing       bool
	err           error
	notice        string

	// Cloudflare sign-in.
	signIn     service.CloudflareSignIn
	accounts   []service.CloudflareAccount
	buckets    []core.Bucket
	step       int
	accountID2 string
	bucket2    string
	readOnly2  bool
	signInBusy bool
	signInErr  error
	status     service.CloudflareStatus
}

// quickState is the image host panel.
type quickState struct {
	open     bool
	source   string
	name     string
	data     []byte
	bitmap   *ui.Bitmap
	width    int
	height   int
	size     int64
	busy     bool
	err      string
	done     bool
	conflict bool
	link     string
	format   string
}

// confirmState is a question asked before something cannot be undone.
type confirmState struct {
	open        bool
	title       string
	message     string
	confirm     string
	destructive bool
	action      func()
}

// transfersState is the transfer page.
type transfersState struct {
	tasks       []core.TransferTask
	filter      string
	rate        float64
	concurrency int
	loaded      bool
}

// usageState is the usage scan.
type usageState struct {
	scanning bool
	usage    core.Usage
	progress core.Usage
	err      error
	location *Location
	cancel   context.CancelFunc
}

// settingsState is the settings page.
type settingsState struct {
	section  int
	settings service.Settings
	clientID string
	redirect string
	scopes   []string
	loaded   bool
	busy     bool
	notice   string
}

// previewState is the Quick Look panel.
type previewState struct {
	open     bool
	key      string
	list     []entry
	info     core.ObjectInfo
	text     core.TextPreview
	busy     bool
	err      error
	link     core.ShareLink
	linkAt   int64
	bitmap   *ui.Bitmap
	fetching bool
}
