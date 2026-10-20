package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Theme choices for the interface.
const (
	themeSystem = "system"
	themeLight  = "light"
	themeDark   = "dark"
)

// Accent names the accent colors the interface offers.
var accentNames = []string{"blue", "indigo", "purple", "pink", "orange", "green", "graphite"}

// Folder view choices.
const (
	viewList    = "list"
	viewGrid    = "grid"
	viewColumns = "columns"
)

// quickPrefs holds the image host defaults, kept apart from the other
// preferences the way the previous interface stored them.
type quickPrefs struct {
	ProfileID string `json:"profileId"`
	Bucket    string `json:"bucket"`
	Template  int    `json:"template"`
	Compress  bool   `json:"compress"`
	WebP      bool   `json:"webp"`
	Format    string `json:"format"`
}

// prefs is the interface state that outlives a run. It holds no credentials.
type prefs struct {
	Theme         string          `json:"theme"`
	Accent        string          `json:"accent"`
	Language      string          `json:"language"`
	View          string          `json:"view"`
	SortKey       string          `json:"sortKey"`
	SortDir       int             `json:"sortDir"`
	Rate          float64         `json:"rate"`
	Concurrency   int             `json:"concurrency"`
	SidebarHidden bool            `json:"sidebarHidden"`
	Collapsed     map[string]bool `json:"collapsed"`
	Quick         quickPrefs      `json:"quick"`

	// ColumnOrder and ColumnWidths keep the file list's columns as the
	// user arranged them.
	ColumnOrder  []string           `json:"columnOrder"`
	ColumnWidths map[string]float32 `json:"columnWidths"`
}

func defaultPrefs() *prefs {
	return &prefs{
		Theme:        themeSystem,
		Accent:       "blue",
		Language:     systemLanguage(),
		View:         viewList,
		SortKey:      "name",
		SortDir:      1,
		Concurrency:  2,
		Collapsed:    map[string]bool{},
		Quick:        quickPrefs{Compress: true, WebP: true, Format: "markdown"},
		ColumnWidths: map[string]float32{},
	}
}

// systemLanguage picks the interface language from the environment.
func systemLanguage() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		if len(value) >= 2 && value[:2] == "zh" {
			return langChinese
		}
		return langEnglish
	}
	return langEnglish
}

// prefsStore loads and saves the preferences file.
type prefsStore struct {
	path string

	mu   sync.Mutex
	last []byte
}

func newPrefsStore(dir string) *prefsStore {
	return &prefsStore{path: filepath.Join(dir, "ui.json")}
}

func (s *prefsStore) load() *prefs {
	p := defaultPrefs()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return p
	}
	s.last = raw
	if err := json.Unmarshal(raw, p); err != nil {
		return defaultPrefs()
	}
	if p.Collapsed == nil {
		p.Collapsed = map[string]bool{}
	}
	if p.ColumnWidths == nil {
		p.ColumnWidths = map[string]float32{}
	}
	if p.Concurrency < 1 {
		p.Concurrency = 2
	}
	return p
}

// save writes the preferences, skipping a write that changes nothing.
func (s *prefsStore) save(p *prefs) error {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if string(raw) == string(s.last) {
		return nil
	}
	s.last = raw
	return os.WriteFile(s.path, raw, 0o600)
}
