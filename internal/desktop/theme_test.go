package desktop

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/ui"
)

func TestPaletteColorsListsEveryColor(t *testing.T) {
	p := newPalette(false, "blue")
	listed := map[uintptr]bool{}
	for _, color := range p.colors() {
		listed[uintptr(unsafe.Pointer(color))] = true
	}
	v := reflect.ValueOf(p).Elem()
	for i := range v.NumField() {
		field := v.Field(i)
		if field.Type() == reflect.TypeFor[ui.Color]() && !listed[field.UnsafeAddr()] {
			t.Errorf("palette.colors misses %s, so it would not fade", v.Type().Field(i).Name)
		}
	}
}

func TestThemeChangeFades(t *testing.T) {
	app := newTestApp(t)
	light, dark := newPalette(false, app.prefs.Accent), newPalette(true, app.prefs.Accent)
	app.palTarget = light
	tt := ui.NewTester(app.view, 1280, 840)
	tt.Frame()
	if app.pal.background != light.background {
		t.Fatalf("first frame background = %v, want %v", app.pal.background, light.background)
	}

	app.palTarget = dark
	tt.Frame()
	tt.Advance(paletteFade / 2)
	halfway := []struct {
		name          string
		got, from, to ui.Color
	}{
		{"background", app.pal.background, light.background, dark.background},
		{"text", app.pal.text, light.text, dark.text},
		{"image tile", app.pal.kindPaletteOf("image").background, light.kindPaletteOf("image").background, dark.kindPaletteOf("image").background},
		{"widget scrollbar", app.pal.widgetTheme().Scrollbar, light.scrollbar, dark.scrollbar},
	}
	for _, color := range halfway {
		if color.got == color.from || color.got == color.to {
			t.Errorf("%s = %v halfway through the fade, want a color between %v and %v",
				color.name, color.got, color.from, color.to)
		}
	}

	tt.Advance(paletteFade)
	if app.pal.background != dark.background || app.pal.text != dark.text {
		t.Errorf("after the fade background, text = %v, %v; want %v, %v",
			app.pal.background, app.pal.text, dark.background, dark.text)
	}
	if got, want := app.pal.kindPaletteOf("image"), dark.kindPaletteOf("image"); got != want {
		t.Errorf("after the fade the image kind = %v, want %v", got, want)
	}
}
