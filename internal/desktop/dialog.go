package desktop

import "github.com/egoist/mygo/ui"

// dialog draws a panel over a scrim. A sheet hangs from the title row over
// the content; any other dialog sits in the middle of the window. Escape
// calls close, and so does a press on the scrim unless the dialog holds a
// form that would lose what was typed. It returns the overlay, for more
// keys of the dialog's own.
func (a *App) dialog(c *ui.Context, width float32, sheet, form bool, close func(), body func()) ui.Element {
	var backdrop ui.Element
	ui.Overlay(c, func() {
		backdrop = ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).Background(a.pal.scrim).Modal()
		if sheet {
			start := float32(0)
			if a.sidebarOpen {
				start = a.sidebarWidth
			}
			backdrop.AlignItems(ui.Start).Padding(titleBarHeight, 0, 16, start)
		} else {
			backdrop.AlignItems(ui.Center).Padding(40)
		}
		backdrop.Children(func() {
			panel := ui.Column(c).Width(width).MaxHeightPercent(100).Clip().
				Background(a.pal.popover).Border(0.5, a.pal.dialogRing).
				Shadow(0, 24, 60, 0, a.pal.dialogShadow)
			if sheet {
				panel.Radius(0, 0, 14, 14)
			} else {
				panel.Radius(14)
			}
			panel.Children(body)
			if !form && panel.PressedOutside() || backdrop.OverlayShortcut(0, ui.KeyEscape) {
				close()
			}
		})
	})
	return backdrop
}

// dialogHeader is a dialog's icon, title and close button. extra goes
// between the title and the button.
func (a *App) dialogHeader(c *ui.Context, glyph *ui.SVG, tile, ink ui.Color, title, subtitle string, extra, close func()) {
	ui.Row(c).Gap(12).Padding(16, 14, 12, 18).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Box(c).Size(34, 34).Radius(10).Center().Shrink(0).Background(tile).Children(func() {
			ui.Icon(c, glyph).FontSize(17).TextColor(ink)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, title).FontSize(15).FontWeight(650).SingleLine()
			if subtitle != "" {
				ui.Text(c, subtitle).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
			}
		})
		if extra != nil {
			extra()
		}
		a.ghostButton(c, iconX, a.t("common.close"), 28, false, close)
	})
}

// dialogFooter is the bar of actions along a dialog's bottom edge.
func (a *App) dialogFooter(c *ui.Context, children func()) {
	ui.Row(c).Gap(8).Padding(12, 20).AlignItems(ui.Center).Shrink(0).
		Background(a.pal.sidebar).BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).
		Children(children)
}

// notice is a line of warning or information inside a dialog or a page.
func (a *App) notice(c *ui.Context, background, color ui.Color, text string) {
	ui.Row(c).Gap(8).Padding(9, 12).Radius(9).AlignItems(ui.Start).
		Background(background).Children(func() {
		ui.Icon(c, iconAlertTriangle).FontSize(13).TextColor(color).Margin(2, 0, 0, 0)
		ui.Text(c, text).FontSize(12.5).TextColor(color).Grow(1).MinWidth(0)
	})
}
