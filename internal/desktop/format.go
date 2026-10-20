package desktop

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// formatBytes writes a size with the unit that fits it.
func formatBytes(value int64) string {
	if value <= 0 {
		return "0 B"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	index := int(math.Log(float64(value)) / math.Log(1024))
	if index >= len(units) {
		index = len(units) - 1
	}
	scaled := float64(value) / math.Pow(1024, float64(index))
	if index == 0 {
		return fmt.Sprintf("%.0f %s", scaled, units[index])
	}
	return trimZero(scaled) + " " + units[index]
}

// trimZero writes a number with at most one decimal place.
func trimZero(value float64) string {
	rounded := math.Round(value*10) / 10
	if rounded == math.Trunc(rounded) {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// formatCount groups thousands with commas.
func formatCount(value int64) string {
	digits := strconv.FormatInt(value, 10)
	negative := strings.HasPrefix(digits, "-")
	if negative {
		digits = digits[1:]
	}
	var out []byte
	for i, digit := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, digit)
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}

// formatDate writes a timestamp for the interface; a zero time is a dash.
func formatDate(value time.Time) string {
	if value.IsZero() || value.Year() <= 1 {
		return "—"
	}
	return value.Format("2006-01-02 15:04")
}

// formatTime writes a shorter timestamp, as the transfer rows show it.
func formatTime(value time.Time) string {
	if value.IsZero() || value.Year() <= 1 {
		return "—"
	}
	return value.Format("Jan 2 15:04")
}

// formatDuration writes seconds as m:ss or h:mm:ss.
func formatDuration(seconds float64) string {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return "—"
	}
	total := int(math.Round(seconds))
	hours := total / 3600
	minutes := (total % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, total%60)
	}
	return fmt.Sprintf("%d:%02d", minutes, total%60)
}

// basename is the last path segment of a key, without its trailing slash.
func basename(key string) string {
	trimmed := strings.TrimSuffix(key, "/")
	if trimmed == "" {
		return key
	}
	if index := strings.LastIndex(trimmed, "/"); index >= 0 {
		return trimmed[index+1:]
	}
	return trimmed
}

// extension is the lower-case extension of a key, without the dot.
func extension(key string) string {
	name := basename(key)
	dot := strings.LastIndex(name, ".")
	if dot <= 0 {
		return ""
	}
	return strings.ToLower(name[dot+1:])
}

// parentPrefix is the prefix holding a key or prefix.
func parentPrefix(prefix string) string {
	trimmed := strings.TrimSuffix(prefix, "/")
	index := strings.LastIndex(trimmed, "/")
	if index < 0 {
		return ""
	}
	return trimmed[:index+1]
}

// crumb is one step of a prefix path.
type crumb struct {
	name   string
	prefix string
}

// breadcrumbs splits a prefix into its steps.
func breadcrumbs(prefix string) []crumb {
	parts := strings.FieldsFunc(prefix, func(r rune) bool { return r == '/' })
	out := make([]crumb, 0, len(parts))
	for i, name := range parts {
		out = append(out, crumb{name: name, prefix: strings.Join(parts[:i+1], "/") + "/"})
	}
	return out
}

// displayType writes the content type, or the upper-case extension.
func displayType(key, contentType string) string {
	mime := strings.TrimSpace(strings.Split(contentType, ";")[0])
	if mime != "" {
		return mime
	}
	if ext := extension(key); ext != "" {
		return strings.ToUpper(ext)
	}
	return "—"
}

// kind names the file category the interface colors and previews by.
type kind = string

var kindByExtension = map[string]kind{
	"png": "image", "jpg": "image", "jpeg": "image", "webp": "image", "gif": "image",
	"avif": "image", "bmp": "image", "ico": "image", "svg": "image", "heic": "image",
	"mp4": "video", "mov": "video", "webm": "video", "m4v": "video", "mkv": "video", "avi": "video",
	"mp3": "audio", "wav": "audio", "m4a": "audio", "ogg": "audio", "flac": "audio",
	"aac": "audio", "opus": "audio",
	"pdf": "pdf",
	"md":  "text", "markdown": "text", "txt": "text", "log": "text", "csv": "text",
	"tsv": "text", "ini": "text", "conf": "text", "env": "text",
	"ts": "code", "tsx": "code", "js": "code", "jsx": "code", "mjs": "code", "cjs": "code",
	"css": "code", "scss": "code", "html": "code", "htm": "code", "xml": "code", "go": "code",
	"rs": "code", "py": "code", "rb": "code", "java": "code", "kt": "code", "swift": "code",
	"c": "code", "h": "code", "cpp": "code", "sh": "code", "sql": "code", "yaml": "code",
	"yml": "code", "toml": "code",
	"json": "json", "jsonl": "json", "ndjson": "json", "geojson": "json",
	"gz": "archive", "tgz": "archive", "zip": "archive", "tar": "archive", "rar": "archive",
	"7z": "archive", "bz2": "archive", "xz": "archive", "zst": "archive", "dmg": "archive",
	"pkg": "archive", "iso": "archive",
	"woff": "font", "woff2": "font", "ttf": "font", "otf": "font",
}

// kindOf names the kind of an object or folder.
func kindOf(key, contentType string, folder bool) kind {
	if folder || strings.HasSuffix(key, "/") {
		return "folder"
	}
	if byExtension, ok := kindByExtension[extension(key)]; ok {
		return byExtension
	}
	mime := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	case mime == "application/pdf":
		return "pdf"
	case strings.Contains(mime, "json"):
		return "json"
	case strings.HasPrefix(mime, "font/"):
		return "font"
	case strings.HasPrefix(mime, "text/"):
		return "text"
	case strings.Contains(mime, "zip"), strings.Contains(mime, "gzip"), strings.Contains(mime, "tar"),
		strings.Contains(mime, "compressed"), strings.Contains(mime, "x-7z"), strings.Contains(mime, "x-rar"):
		return "archive"
	}
	return "other"
}

// kindOfExtension names the kind of an extension key from a usage scan,
// such as ".png" or "(none)".
func kindOfExtension(ext string) kind {
	if !strings.HasPrefix(ext, ".") {
		return "other"
	}
	if byExtension, ok := kindByExtension[ext[1:]]; ok {
		return byExtension
	}
	return "other"
}

// previewMode is how the preview shows a kind, or "none" when it cannot.
func previewMode(k kind) string {
	switch k {
	case "image", "video", "audio", "pdf":
		return k
	case "text", "code", "json":
		return "text"
	}
	return "none"
}

// iconOf names the icon a kind shows with.
func iconOf(k kind) *ui.SVG {
	switch k {
	case "folder":
		return iconFolder
	case "image":
		return iconImage
	case "video":
		return iconVideo
	case "audio":
		return iconMusic
	case "pdf":
		return iconFileText
	case "text":
		return iconFileText
	case "code":
		return iconCode
	case "json":
		return iconBraces
	case "archive":
		return iconArchive
	case "font":
		return iconType
	}
	return iconFile
}

// entry is one row of the file browser: an object, or a common prefix
// standing for a folder.
type entry struct {
	key         string
	name        string
	folder      bool
	size        int64
	modified    time.Time
	contentType string
	etag        string
	metadata    map[string]string
}

func (e entry) kind() kind { return kindOf(e.key, e.contentType, e.folder) }

// objectEntry makes a row of an object.
func objectEntry(object core.ObjectInfo, name string) entry {
	return entry{
		key:         object.Key,
		name:        name,
		size:        object.Size,
		modified:    object.LastModified,
		contentType: object.ContentType,
		etag:        object.ETag,
		metadata:    object.Metadata,
	}
}

// folderEntry makes a row of a common prefix.
func folderEntry(key string) entry {
	return entry{key: key, name: basename(key), folder: true}
}

// entriesFromPages turns listing pages into rows, hiding the folder marker
// object of the prefix being listed.
func entriesFromPages(pages []core.ListPage, prefix string) []entry {
	seen := map[string]bool{}
	out := []entry{}
	for _, page := range pages {
		for _, p := range page.Prefixes {
			if seen[p] || p == prefix {
				continue
			}
			seen[p] = true
			out = append(out, folderEntry(p))
		}
		for _, object := range page.Objects {
			if object.Key == prefix || seen[object.Key] {
				continue
			}
			seen[object.Key] = true
			if strings.HasSuffix(object.Key, "/") {
				out = append(out, folderEntry(object.Key))
				continue
			}
			out = append(out, objectEntry(object, basename(object.Key)))
		}
	}
	return out
}

// entriesFromFind turns search hits into rows that show their path below
// the searched prefix.
func entriesFromFind(objects []core.ObjectInfo, prefix string) []entry {
	out := make([]entry, 0, len(objects))
	for _, object := range objects {
		if object.Key == prefix || strings.HasSuffix(object.Key, "/") {
			continue
		}
		name := strings.TrimPrefix(object.Key, prefix)
		if name == "" {
			name = basename(object.Key)
		}
		out = append(out, objectEntry(object, name))
	}
	return out
}

// sortEntries orders rows by a column, keeping folders first.
func sortEntries(rows []entry, key string, direction int) []entry {
	out := append([]entry(nil), rows...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.folder != b.folder {
			return a.folder
		}
		compare := 0
		switch key {
		case "size":
			switch {
			case a.size < b.size:
				compare = -1
			case a.size > b.size:
				compare = 1
			}
		case "modified":
			switch {
			case a.modified.Before(b.modified):
				compare = -1
			case a.modified.After(b.modified):
				compare = 1
			}
		}
		if compare == 0 {
			compare = naturalCompare(a.name, b.name)
		}
		if direction < 0 {
			return compare > 0
		}
		return compare < 0
	})
	return out
}

// naturalCompare orders names with numbers read as numbers, as Finder does.
func naturalCompare(a, b string) int {
	lowerA, lowerB := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(lowerA) && j < len(lowerB) {
		digitA, digitB := isDigit(lowerA[i]), isDigit(lowerB[j])
		switch {
		case digitA && digitB:
			startA, startB := i, j
			for i < len(lowerA) && isDigit(lowerA[i]) {
				i++
			}
			for j < len(lowerB) && isDigit(lowerB[j]) {
				j++
			}
			numberA := strings.TrimLeft(lowerA[startA:i], "0")
			numberB := strings.TrimLeft(lowerB[startB:j], "0")
			if len(numberA) != len(numberB) {
				return sign(len(numberA) - len(numberB))
			}
			if numberA != numberB {
				return sign(strings.Compare(numberA, numberB))
			}
		case lowerA[i] == lowerB[j]:
			i++
			j++
		default:
			return sign(int(lowerA[i]) - int(lowerB[j]))
		}
	}
	return sign((len(lowerA) - i) - (len(lowerB) - j))
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func sign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	}
	return 0
}
