package desktop

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/egoist/mygo/ui"
)

//go:embed icons/*.svg
var iconFS embed.FS

// The icon set is Lucide, drawn as outlines in the color of the text; see
// icons/LICENSE.
var iconSet = loadIcons()

func loadIcons() map[string]*ui.SVG {
	names, err := fs.Glob(iconFS, "icons/*.svg")
	if err != nil {
		panic("desktop: " + err.Error())
	}
	set := make(map[string]*ui.SVG, len(names))
	for _, name := range names {
		data, err := iconFS.ReadFile(name)
		if err != nil {
			panic("desktop: " + err.Error())
		}
		key := strings.TrimSuffix(strings.TrimPrefix(name, "icons/"), ".svg")
		set[key] = ui.MustParseSVG(data)
	}
	return set
}

// icon returns a named icon. An unknown name is a programming error, so it
// panics at start-up rather than drawing nothing.
func icon(name string) *ui.SVG {
	svg, ok := iconSet[name]
	if !ok {
		panic(fmt.Sprintf("desktop: no icon named %q", name))
	}
	return svg
}

var (
	iconAlertTriangle   = icon("alert-triangle")
	iconArchive         = icon("archive")
	iconArrowDownToLine = icon("arrow-down-to-line")
	iconArrowDownUp     = icon("arrow-down-up")
	iconArrowRightLeft  = icon("arrow-right-left")
	iconArrowUpFromLine = icon("arrow-up-from-line")
	iconArrowUpRight    = icon("arrow-up-right")
	iconBot             = icon("bot")
	iconBraces          = icon("braces")
	iconCheck           = icon("check")
	iconChevronDown     = icon("chevron-down")
	iconChevronLeft     = icon("chevron-left")
	iconChevronRight    = icon("chevron-right")
	iconCircleAlert     = icon("circle-alert")
	iconClock           = icon("clock")
	iconCode            = icon("code")
	iconColumns         = icon("columns-3")
	iconCopy            = icon("copy")
	iconDatabase        = icon("database")
	iconDownload        = icon("download")
	iconFile            = icon("file")
	iconFileText        = icon("file-text")
	iconFolder          = icon("folder")
	iconGlobe           = icon("globe")
	iconImage           = icon("image")
	iconImagePlus       = icon("image-plus")
	iconInfo            = icon("info")
	iconKeyRound        = icon("key-round")
	iconKeyboard        = icon("keyboard")
	iconLayoutGrid      = icon("layout-grid")
	iconLink            = icon("link")
	iconList            = icon("list")
	iconLock            = icon("lock")
	iconMusic           = icon("music")
	iconPalette         = icon("palette")
	iconPanelLeft       = icon("panel-left")
	iconPause           = icon("pause")
	iconPieChart        = icon("pie-chart")
	iconPlay            = icon("play")
	iconPlug            = icon("plug")
	iconPlus            = icon("plus")
	iconRefresh         = icon("refresh-cw")
	iconScale           = icon("scale")
	iconSearch          = icon("search")
	iconSearchX         = icon("search-x")
	iconSettings        = icon("settings-2")
	iconShare           = icon("share-2")
	iconShieldCheck     = icon("shield-check")
	iconShieldX         = icon("shield-x")
	iconSliders         = icon("sliders-horizontal")
	iconSquareTerminal  = icon("square-terminal")
	iconTrash           = icon("trash-2")
	iconType            = icon("type")
	iconUpload          = icon("upload")
	iconVideo           = icon("video")
	iconWifiOff         = icon("wifi-off")
	iconX               = icon("x")
	iconZap             = icon("zap")
)
