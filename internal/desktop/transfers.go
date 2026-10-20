package desktop

import (
	"sort"
	"strconv"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo/ui"
)

// Filters of the transfers page.
const (
	filterAll       = "all"
	filterActive    = "active"
	filterDone      = "done"
	filterAttention = "attention"
)

// rates are the speed limits the page offers, in MiB/s.
var rates = []float64{0, 1, 2, 5, 10, 20, 50}

// statusOrder puts the transfers that need attention first.
var statusOrder = map[string]int{
	"running": 0, "cancelling": 1, "queued": 2, "paused": 3,
	"failed": 4, "completed": 5, "cancelled": 6,
}

// concurrencies are the numbers of transfers the page lets run at once.
var concurrencies = []int{1, 2, 4, 8}

// transfersPage draws the transfer queue as a table.
func (a *App) transfersPage(c *ui.Context) {
	a.applyTransferSettings()
	tasks := a.filteredTasks()
	ui.Column(c).Fill().Children(func() {
		a.transfersHeader(c)
		if len(tasks) == 0 {
			a.transfersEmpty(c)
			return
		}
		ui.Scroll(c).Grow(1).Children(func() {
			for i := range tasks {
				a.transferRow(c, &tasks[i])
			}
		})
	})
}

// applyTransferSettings pushes the saved speed limit and concurrency once.
func (a *App) applyTransferSettings() {
	if a.transfers.loaded {
		return
	}
	a.transfers.loaded = true
	a.transfers.rate = a.prefs.Rate
	a.transfers.concurrency = a.prefs.Concurrency
	go func() {
		concurrency := a.svc.Transfers.SetConcurrency(a.prefs.Concurrency)
		_ = concurrency
		a.svc.Transfers.SetRate(int64(a.prefs.Rate * 1024 * 1024))
	}()
}

// transfersToolbar is the rest of the page's title row: the counts, the
// filter and the two queue settings.
func (a *App) transfersToolbar(c *ui.Context) {
	running, queued, speed := 0, 0, int64(0)
	for _, task := range a.transfers.tasks {
		switch task.Status {
		case "running":
			running++
			speed += task.BytesPerSecond
		case "queued":
			queued++
		}
	}
	summary := a.tf("transfers.summary", "running", running, "queued", queued, "speed", formatBytes(speed))
	ui.Text(c, summary).FontSize(12).TextColor(a.pal.muted).FontFeatures("tnum").
		SingleLine().Grow(1).MinWidth(0)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
		a.filterBar(c)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.t("transfers.concurrency")).FontSize(12).TextColor(a.pal.muted).SingleLine()
			a.concurrencySwitch(c)
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.t("transfers.limit")).FontSize(12).TextColor(a.pal.muted).SingleLine()
			a.rateStepper(c, 26)
			unit := a.t("transfers.perSecond")
			if a.prefs.Rate == 0 {
				unit = a.t("transfers.unlimited")
			}
			ui.Text(c, unit).FontSize(12).TextColor(a.pal.muted).SingleLine()
		})
	})
}

// concurrencySwitch chooses how many transfers run at once.
func (a *App) concurrencySwitch(c *ui.Context) {
	labels := make([]string, len(concurrencies))
	index := -1
	for i, value := range concurrencies {
		labels[i] = strconv.Itoa(value)
		if value == a.prefs.Concurrency {
			index = i
		}
	}
	if !a.segments(c, &index, a.t("transfers.concurrency"), labels) {
		return
	}
	if index < 0 || index >= len(concurrencies) {
		return
	}
	value := concurrencies[index]
	a.prefs.Concurrency = value
	a.savePrefs()
	go func() { a.svc.Transfers.SetConcurrency(value) }()
}

// rateStepper steps the speed limit through rates; 0 is no limit.
func (a *App) rateStepper(c *ui.Context, height float32) {
	index := len(rates) - 1
	for i, rate := range rates {
		if rate >= a.prefs.Rate {
			index = i
			break
		}
	}
	ui.Row(c).Height(height).Radius(7).Border(1, a.pal.borderStrong).Clip().
		AlignItems(ui.Center).Shrink(0).Children(func() {
		if a.stepButton(c, "−", a.t("transfers.slower"), height, index == 0) {
			a.setRate(rates[index-1])
		}
		ui.Text(c, trimZero(a.prefs.Rate)).FontSize(12).FontFeatures("tnum").
			MinWidth(34).TextAlign(ui.Center).TextColor(a.pal.text)
		if a.stepButton(c, "+", a.t("transfers.faster"), height, index == len(rates)-1) {
			a.setRate(rates[index+1])
		}
	})
}

// stepButton is one end of a stepper.
func (a *App) stepButton(c *ui.Context, glyph, label string, height float32, disabled bool) bool {
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(height, height).Disabled(disabled)
	color := a.pal.muted
	if disabled {
		color = a.pal.faint
	} else if button.Hovered() {
		button.Background(a.pal.hover)
		color = a.pal.text
	}
	button.Children(func() { ui.Text(c, glyph).FontSize(14).TextColor(color) })
	return !disabled && button.Clicked()
}

// setRate saves the speed limit, in MiB/s, and applies it.
func (a *App) setRate(rate float64) {
	a.prefs.Rate = rate
	a.savePrefs()
	go func(bytes int64) { a.svc.Transfers.SetRate(bytes) }(int64(rate * 1024 * 1024))
}

// filteredTasks orders the queue for the chosen filter.
func (a *App) filteredTasks() []core.TransferTask {
	keep := func(task core.TransferTask) bool {
		switch a.transfers.filter {
		case filterActive:
			return isActive(task)
		case filterDone:
			return task.Status == "completed"
		case filterAttention:
			return task.Status == "failed" || task.Status == "paused"
		}
		return true
	}
	out := make([]core.TransferTask, 0, len(a.transfers.tasks))
	for _, task := range a.transfers.tasks {
		if keep(task) {
			out = append(out, task)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, right := statusOrder[out[i].Status], statusOrder[out[j].Status]
		if left != right {
			return left < right
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// filterBar chooses which transfers are listed.
func (a *App) filterBar(c *ui.Context) {
	filters := []string{filterAll, filterActive, filterDone, filterAttention}
	labels := []string{
		a.t("transfers.all"), a.t("transfers.active"),
		a.t("transfers.done"), a.t("transfers.attention"),
	}
	index := 0
	for i, name := range filters {
		if name == a.transfers.filter {
			index = i
		}
	}
	if a.segments(c, &index, a.t("transfers.title"), labels) && index >= 0 && index < len(filters) {
		a.transfers.filter = filters[index]
	}
}

// The columns of the transfer table, past the icon: name and progress
// share the room left by the fixed ones.
const (
	transferNameShare     = 1.6
	transferProgressShare = 1.2
	transferStatusWidth   = 96
	transferSpeedWidth    = 118
	transferActionsWidth  = 96
)

// transferColumns lays out one line of the table.
func (a *App) transferColumns(c *ui.Context, row ui.Element, lead, name, progress, status, speed, actions func()) {
	row.Gap(14).PaddingX(20).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(28).Shrink(0).Children(lead)
		ui.Column(c).Grow(transferNameShare).MinWidth(0).Children(name)
		ui.Column(c).Grow(transferProgressShare).MinWidth(0).Children(progress)
		ui.Row(c).Width(transferStatusWidth).Shrink(0).Children(status)
		ui.Column(c).Width(transferSpeedWidth).Shrink(0).Children(speed)
		ui.Row(c).Width(transferActionsWidth).Shrink(0).Gap(2).Justify(ui.End).Children(actions)
	})
}

// transfersHeader names the columns.
func (a *App) transfersHeader(c *ui.Context) {
	label := func(key string) func() {
		return func() {
			ui.Text(c, a.t(key)).FontSize(11.5).FontWeight(500).TextColor(a.pal.muted).SingleLine()
		}
	}
	row := ui.Row(c).Padding(8, 0).BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border)
	a.transferColumns(c, row, func() {}, label("transfers.colName"), label("transfers.colProgress"),
		label("transfers.colStatus"), label("transfers.colSpeed"), func() {})
}

// transfersEmpty explains an empty queue.
func (a *App) transfersEmpty(c *ui.Context) {
	ui.Column(c).Padding(90, 20).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(56, 56).Radius(16).Center().Background(a.pal.mutedFill).Children(func() {
			ui.Icon(c, iconArrowDownUp).FontSize(28).TextColor(a.pal.muted)
		})
		text := a.t("transfers.empty")
		if a.transfers.filter != filterAll {
			text = a.t("transfers.emptyFiltered")
		}
		ui.Text(c, text).FontSize(13).TextColor(a.pal.muted)
	})
}

// transferRow is one transfer with its progress and actions.
func (a *App) transferRow(c *ui.Context, task *core.TransferTask) {
	row := ui.Row(c).Padding(12, 0).BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border)
	if row.Hovered() {
		row.Background(a.pal.hover)
	}
	lead := func() {
		glyph, background, color := iconArrowDownToLine, a.pal.successBg, a.pal.successText
		if task.Direction == "upload" {
			glyph, background, color = iconArrowUpFromLine, a.pal.primarySoft, a.pal.primaryText
		}
		ui.Box(c).Size(28, 28).Radius(8).Center().Background(background).Children(func() {
			ui.Icon(c, glyph).FontSize(14).TextColor(color)
		})
	}
	name := func() {
		ui.Text(c, basename(task.Object.Key)).FontSize(13).FontWeight(500).SingleLine()
		ui.Text(c, a.transferPlace(task)).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
		if task.Error != nil && (task.Status == "failed" || task.Status == "paused") {
			color := a.pal.destructive
			if task.Status == "paused" {
				color = a.pal.warningText
			}
			ui.Text(c, a.errorText(task.Error)).FontSize(11.5).TextColor(color).Margin(3, 0, 0, 0).MaxLines(2)
		}
	}
	progress := func() {
		ui.Column(c).Gap(5).Children(func() {
			a.progressBar(c, progressOf(*task), 5, a.transferTone(task))
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.transferBytes(task)).FontSize(11.5).TextColor(a.pal.muted).
					FontFeatures("tnum").SingleLine().Grow(1)
				ui.Text(c, a.transferParts(task)).FontSize(11.5).TextColor(a.pal.muted).
					FontFeatures("tnum").SingleLine()
			})
		})
	}
	status := func() { a.transferStatus(c, task) }
	speed := func() {
		lines := a.transferSpeed(task)
		for _, line := range lines {
			ui.Text(c, line).FontSize(11.5).TextColor(a.pal.muted).FontFeatures("tnum").SingleLine()
		}
	}
	actions := func() { a.transferActions(c, task) }
	a.transferColumns(c, row, lead, name, progress, status, speed, actions)
}

// transferPlace names the connection and the folder of a transfer.
func (a *App) transferPlace(task *core.TransferTask) string {
	place := task.Object.Bucket + "/" + parentPrefix(task.Object.Key)
	if profile := a.profileOf(task.Object.ProfileID); profile != nil {
		place = profile.Name + " · " + place
	}
	return place
}

// transferTone colors the progress bar of a transfer by its state.
func (a *App) transferTone(task *core.TransferTask) ui.Color {
	switch task.Status {
	case "completed":
		return a.pal.success
	case "failed":
		return a.pal.destructive
	case "paused":
		return a.pal.warning
	case "cancelled", "cancelling":
		return a.pal.faint
	}
	return a.pal.primary
}

// transferStatus is the chip naming what a transfer is doing.
func (a *App) transferStatus(c *ui.Context, task *core.TransferTask) {
	label := a.t("transfers.status." + task.Status)
	background, color := a.pal.mutedFill, a.pal.muted
	switch task.Status {
	case "running":
		background, color = a.pal.primarySoft, a.pal.primaryText
	case "completed":
		background, color = a.pal.successBg, a.pal.successText
	case "failed":
		background, color = a.pal.destructiveBg, a.pal.destructive
	case "paused":
		background, color = a.pal.warningBg, a.pal.warningText
	case "cancelled":
		color = a.pal.faint
	}
	ui.Row(c).Height(20).PaddingX(8).Gap(5).Radius(10).AlignItems(ui.Center).
		Background(background).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(color)
		ui.Text(c, label).FontSize(11).FontWeight(600).TextColor(color).SingleLine()
	})
}

// transferSpeed writes the speed and the remaining time, a dash when the
// transfer is not moving.
func (a *App) transferSpeed(task *core.TransferTask) []string {
	if task.Status != "running" || task.BytesPerSecond <= 0 {
		return []string{"—"}
	}
	remaining := float64(task.Size-task.Completed) / float64(task.BytesPerSecond)
	return []string{
		formatBytes(task.BytesPerSecond) + "/s",
		a.tf("transfers.left", "time", formatDuration(remaining)),
	}
}

// transferBytes writes how much of a transfer is done.
func (a *App) transferBytes(task *core.TransferTask) string {
	if task.Size > 0 {
		return formatBytes(task.Completed) + " / " + formatBytes(task.Size)
	}
	return ""
}

// transferParts counts the parts of a multipart transfer.
func (a *App) transferParts(task *core.TransferTask) string {
	if task.PartSize <= 0 || task.Size <= task.PartSize {
		return ""
	}
	total := (task.Size + task.PartSize - 1) / task.PartSize
	return a.tf("transfers.parts", "done", task.Completed/task.PartSize, "total", total)
}

// transferActions offers what a transfer's state allows.
func (a *App) transferActions(c *ui.Context, task *core.TransferTask) {
	resume := func() error {
		_, err := a.svc.ResumeTransfer(a.ctx, task.ID)
		return err
	}
	switch task.Status {
	case "running":
		if a.transferButton(c, iconPause, a.t("transfers.pause"), false) {
			a.transferCommand(func() error { return a.svc.PauseTransfer(a.ctx, task.ID) })
		}
	case "paused":
		if a.transferButton(c, iconPlay, a.t("transfers.resume"), true) {
			a.transferCommand(resume)
		}
	case "failed", "cancelled":
		if a.transferButton(c, iconRefresh, a.t("transfers.retry"), true) {
			a.transferCommand(resume)
		}
	}
	if task.Status == "queued" || task.Status == "running" || task.Status == "paused" {
		if a.transferButton(c, iconX, a.t("transfers.cancel"), false) {
			a.askCancelTransfer(task.ID)
		}
	}
}

// transferButton is an action of a transfer row. An accented one moves the
// transfer forward.
func (a *App) transferButton(c *ui.Context, glyph *ui.SVG, label string, accent bool) bool {
	if !accent {
		clicked := false
		a.ghostButton(c, glyph, label, 28, false, func() { clicked = true })
		return clicked
	}
	button := a.plainButton(c, label, 28, 28)
	if button.Hovered() {
		button.Background(a.pal.primarySoft)
	}
	button.Children(func() { ui.Icon(c, glyph).FontSize(15).TextColor(a.pal.primaryText) })
	return button.Clicked()
}

// iconAction is a small icon-only button, flat until the pointer is on it.
func (a *App) iconAction(c *ui.Context, glyph *ui.SVG, label string) bool {
	clicked := false
	a.ghostButton(c, glyph, label, 26, false, func() { clicked = true })
	return clicked
}

// transferCommand runs a queue command and refreshes the page.
func (a *App) transferCommand(action func() error) {
	go func() {
		err := action()
		a.update(func() {
			if err != nil {
				a.toastCurrent(a.errorText(err))
			}
			a.loadTasks()
		})
	}()
}

// askCancelTransfer confirms before a transfer is stopped.
func (a *App) askCancelTransfer(id string) {
	a.confirm = confirmState{
		open:    true,
		title:   a.t("transfers.cancel"),
		message: a.t("transfers.cancelConfirm"),
		confirm: a.t("transfers.cancel"),
		action: func() {
			a.transferCommand(func() error { return a.svc.CancelTransfer(a.ctx, id) })
		},
	}
}

// isActive reports whether a transfer is still moving.
func isActive(task core.TransferTask) bool {
	switch task.Status {
	case "queued", "running", "cancelling":
		return true
	}
	return false
}

// progressOf is how much of a transfer is done, from 0 to 1.
func progressOf(task core.TransferTask) float64 {
	if task.Size <= 0 {
		return 0
	}
	ratio := float64(task.Completed) / float64(task.Size)
	if ratio > 1 {
		return 1
	}
	return ratio
}
