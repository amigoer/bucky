package desktop

import "github.com/egoist/mygo/ui"

// welcome is the page shown before anything is open.
func (a *App) welcome(c *ui.Context) {
	ui.Column(c).Fill().Center().Padding(32).Children(func() {
		ui.Column(c).MaxWidth(420).Gap(10).AlignItems(ui.Center).Children(func() {
			a.logoMark(c, 64)
			ui.Text(c, a.t("welcome.title")).FontSize(22).Bold().TextAlign(ui.Center)
			ui.Text(c, a.t("welcome.desc")).FontSize(12.5).TextColor(a.pal.muted).
				TextAlign(ui.Center)
			ui.Row(c).Gap(8).Margin(6, 0, 0, 0).Children(func() {
				if ui.PrimaryButton(c, a.t("welcome.add")).Clicked() {
					a.openConnect("")
				}
			})
			if len(a.profiles) > 0 {
				ui.Text(c, a.t("welcome.pick")).FontSize(12).TextColor(a.pal.faint).
					TextAlign(ui.Center)
			}
		})
	})
}
