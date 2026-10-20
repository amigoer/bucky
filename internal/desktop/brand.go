package desktop

import "github.com/egoist/mygo/ui"

// The application mark is a bucket, its body and its rim drawn apart so the
// rim can be lighter.
var (
	bucketBody = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><path d="M5 6h10l-1.6 9H6.6z" fill="currentColor"/></svg>`))
	bucketRim  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><rect x="4" y="4.5" width="12" height="2" rx="1" fill="currentColor"/></svg>`))
)

// logoMark draws the application mark on a tile of the accent color.
func (a *App) logoMark(c *ui.Context, size float32) {
	white := ui.Hex("#ffffff")
	ui.Box(c).Size(size, size).Radius(size*0.3).Clip().
		LinearGradient(ui.LinearGradient{From: a.pal.logoFrom, To: a.pal.logoTo, Angle: 160, Oklab: true}).
		InsetShadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.12)).Children(func() {
		ui.Icon(c, bucketBody).FontSize(size).TextColor(white).Absolute().Left(0).Top(0)
		ui.Icon(c, bucketRim).FontSize(size).TextColor(white).Opacity(0.75).Absolute().Left(0).Top(0)
	})
}

// brand draws the application mark and its name.
func (a *App) brand(c *ui.Context) {
	ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
		a.logoMark(c, 20)
		ui.Text(c, "Bucky").FontSize(14).FontWeight(700).LetterSpacing(-0.28)
	})
}
