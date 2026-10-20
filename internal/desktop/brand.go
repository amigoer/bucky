package desktop

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/logo.png
var logoPNG []byte

var logo *ui.Bitmap

func appLogo() *ui.Bitmap {
	if logo != nil {
		return logo
	}
	img, _, err := image.Decode(bytes.NewReader(logoPNG))
	if err != nil {
		return nil
	}
	logo = ui.NewBitmap(img)
	return logo
}

// brand draws the application mark and its name.
func (a *App) brand(c *ui.Context) {
	ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
		if mark := appLogo(); mark != nil {
			ui.Image(c, mark).Size(20, 20).Radius(6)
		}
		ui.Text(c, "Bucky").FontSize(13.5).Bold()
	})
}
