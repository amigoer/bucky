package desktop

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales/*.json
var localeFS embed.FS

// Languages the desktop application ships with.
const (
	langEnglish = "en"
	langChinese = "zh"
)

// messages resolves the interface strings of one language. The catalogs are
// nested JSON, flattened to dotted keys once at start-up.
type messages struct {
	lang string
	text map[string]string
}

func loadMessages(lang string) *messages {
	if lang != langChinese {
		lang = langEnglish
	}
	m := &messages{lang: lang, text: map[string]string{}}
	for _, name := range []string{langEnglish, lang} {
		raw, err := localeFS.ReadFile("locales/" + name + ".json")
		if err != nil {
			continue
		}
		var tree map[string]any
		if err := json.Unmarshal(raw, &tree); err != nil {
			continue
		}
		flatten(m.text, "", tree)
	}
	return m
}

func flatten(dst map[string]string, prefix string, tree map[string]any) {
	for key, value := range tree {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		switch typed := value.(type) {
		case string:
			dst[full] = typed
		case map[string]any:
			flatten(dst, full, typed)
		}
	}
}

// t returns a string, or its key when the catalog has none.
func (m *messages) t(key string) string {
	if text, ok := m.text[key]; ok {
		return text
	}
	return key
}

// tf returns a string with {name} placeholders replaced.
func (m *messages) tf(key string, args ...any) string {
	return substitute(m.t(key), args...)
}

// tn picks the singular or plural form of a counted string. A string
// without plural forms uses the count placeholder in its only form.
func (m *messages) tn(key string, count int, args ...any) string {
	suffix := "_other"
	if count == 1 {
		suffix = "_one"
	}
	text, ok := m.text[key+suffix]
	if !ok {
		text, ok = m.text[key+"_other"]
	}
	if !ok {
		text = m.t(key)
	}
	return substitute(text, append([]any{"count", count}, args...)...)
}

// substitute replaces {name} with the value of each name/value pair.
func substitute(text string, args ...any) string {
	if len(args) < 2 {
		return text
	}
	pairs := make([]string, 0, len(args))
	for i := 0; i+1 < len(args); i += 2 {
		name, ok := args[i].(string)
		if !ok {
			continue
		}
		pairs = append(pairs, "{{"+name+"}}", fmt.Sprint(args[i+1]))
	}
	return strings.NewReplacer(pairs...).Replace(text)
}
