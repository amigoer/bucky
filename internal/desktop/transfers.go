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

// transfersPage draws the transfer queue.
func (a *App) transfersPage(c *ui.Context) {
	a.applyTransferSettings()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).MaxWidth(920).Gap(16).Padding(20, 24).Margin(0, ui.Auto).Children(func() {
			a.transfersSummary(c)
			a.transfersTable(c)
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

// transfersSummary is the line of counts and the two queue settings.
func (a *App) transfersSummary(c *ui.Context) {
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
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.tf("transfers.summary", "running", running, "queued", queued,
			"speed", formatBytes(speed))).FontSize(12).TextColor(a.pal.muted)
		ui.Spacer(c)
		ui.Text(c, a.t("transfers.concurrency")).FontSize(11).TextColor(a.pal.faint)
		a.concurrencySelect(c)
		ui.Text(c, a.t("transfers.limit")).FontSize(11).TextColor(a.pal.faint)
		a.rateSelect(c)
	})
}

func (a *App) concurrencySelect(c *ui.Context) {
	options := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	current := strconv.Itoa(a.prefs.Concurrency)
	select_ := ui.Select(c, &current, options).Label(a.t("transfers.concurrency")).Width(72)
	if !select_.Changed() {
		return
	}
	value, err := strconv.Atoi(current)
	if err != nil || value < 1 {
		return
	}
	a.prefs.Concurrency = value
	a.savePrefs()
	go func() { a.svc.Transfers.SetConcurrency(value) }()
}

func (a *App) rateSelect(c *ui.Context) {
	options := make([]string, 0, len(rates))
	for _, rate := range rates {
		if rate == 0 {
			options = append(options, a.t("transfers.unlimited"))
			continue
		}
		options = append(options, trimZero(rate)+" "+a.t("transfers.perSecond"))
	}
	current := options[0]
	for i, rate := range rates {
		if rate == a.prefs.Rate {
			current = options[i]
		}
	}
	select_ := ui.Select(c, &current, options).Label(a.t("transfers.limit")).Width(124)
	if !select_.Changed() {
		return
	}
	for i, option := range options {
		if option == current {
			a.prefs.Rate = rates[i]
			a.savePrefs()
			go func(bytes int64) { a.svc.Transfers.SetRate(bytes) }(int64(rates[i] * 1024 * 1024))
			return
		}
	}
}

// transfersTable lists the transfers of the current filter.
func (a *App) transfersTable(c *ui.Context) {
	tasks := a.filteredTasks()
	ui.Column(c).Gap(10).Children(func() {
		a.filterBar(c)
		if len(tasks) == 0 {
			a.transfersEmpty(c)
			return
		}
		ui.Column(c).Gap(1).Children(func() {
			for i := range tasks {
				a.transferRow(c, &tasks[i])
			}
		})
	})
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
	seg := ui.Segmented(c, &index, labels...).Label(a.t("transfers.title")).Width(320)
	if seg.Changed() && index >= 0 && index < len(filters) {
		a.transfers.filter = filters[index]
	}
}

// transfersEmpty explains an empty queue.
func (a *App) transfersEmpty(c *ui.Context) {
	ui.Column(c).Padding(48).Center().Children(func() {
		ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconArrowDownUp).FontSize(28).TextColor(a.pal.faint)
			text := a.t("transfers.empty")
			if a.transfers.filter != filterAll {
				text = a.t("transfers.emptyFiltered")
			}
			ui.Text(c, text).FontSize(12.5).TextColor(a.pal.muted)
		})
	})
}

// transferRow is one transfer with its progress and actions.
func (a *App) transferRow(c *ui.Context, task *core.TransferTask) {
	upload := task.Direction == "upload"
	ui.Column(c).Gap(6).Padding(10, 12).Radius(10).
		Background(a.pal.background).Border(1, a.pal.border).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			glyph := iconArrowDownToLine
			if upload {
				glyph = iconArrowUpFromLine
			}
			ui.Box(c).Size(28, 28).Radius(7).Center().Background(a.pal.tile).Children(func() {
				ui.Icon(c, glyph).FontSize(15).TextColor(a.pal.muted)
			})
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, basename(task.Object.Key)).FontSize(12.5).Bold().SingleLine()
				ui.Text(c, task.Object.Bucket).FontSize(11).TextColor(a.pal.faint).SingleLine()
			})
			a.transferStatus(c, task)
			ui.Column(c).Width(120).AlignItems(ui.End).Children(func() {
				ui.Text(c, a.transferSpeed(task)).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
				ui.Text(c, a.transferParts(task)).FontSize(11).TextColor(a.pal.faint).SingleLine()
			})
			a.transferActions(c, task)
		})
		a.progressBar(c, progressOf(*task), 4, a.transferTone(task))
	})
}

// transferTone colors the progress bar of a transfer by its state.
func (a *App) transferTone(task *core.TransferTask) ui.Color {
	switch task.Status {
	case "completed":
		return a.pal.success
	case "failed", "cancelled":
		return a.pal.destructive
	case "paused":
		return a.pal.warning
	}
	return a.pal.primary
}

// transferStatus is the badge naming what a transfer is doing.
func (a *App) transferStatus(c *ui.Context, task *core.TransferTask) {
	label := a.t("transfers.status." + task.Status)
	background, color := a.pal.field, a.pal.muted
	switch task.Status {
	case "running":
		background, color = a.pal.primarySoft, a.pal.primaryText
	case "completed":
		background, color = a.pal.successBg, a.pal.successText
	case "failed", "cancelled":
		background, color = a.pal.destructiveBg, a.pal.destructive
	case "paused":
		background, color = a.pal.warningBg, a.pal.warningText
	}
	ui.Badge(c, label).Background(background).TextColor(color)
}

// transferSpeed writes the speed and the remaining time.
func (a *App) transferSpeed(task *core.TransferTask) string {
	if task.Status != "running" || task.BytesPerSecond <= 0 {
		return ""
	}
	remaining := float64(task.Size-task.Completed) / float64(task.BytesPerSecond)
	return formatBytes(task.BytesPerSecond) + "/s · " +
		a.tf("transfers.left", "time", formatDuration(remaining))
}

// transferParts writes the part count, or the size for a whole-object transfer.
func (a *App) transferParts(task *core.TransferTask) string {
	if task.Size > 0 {
		return formatBytes(task.Completed) + " / " + formatBytes(task.Size)
	}
	return ""
}

// transferActions offers what a transfer's state allows.
func (a *App) transferActions(c *ui.Context, task *core.TransferTask) {
	ui.Row(c).Gap(2).Children(func() {
		switch task.Status {
		case "running":
			if a.iconAction(c, iconPause, a.t("transfers.pause")) {
				a.transferCommand(func() error { return a.svc.PauseTransfer(a.ctx, task.ID) })
			}
		case "paused":
			if a.iconAction(c, iconPlay, a.t("transfers.resume")) {
				a.transferCommand(func() error {
					_, err := a.svc.ResumeTransfer(a.ctx, task.ID)
					return err
				})
			}
		case "failed", "cancelled":
			if a.iconAction(c, iconRefresh, a.t("transfers.retry")) {
				a.transferCommand(func() error {
					_, err := a.svc.ResumeTransfer(a.ctx, task.ID)
					return err
				})
			}
		}
		if task.Status == "queued" || task.Status == "running" || task.Status == "paused" ||
			task.Status == "cancelling" {
			if a.iconAction(c, iconX, a.t("transfers.cancel")) {
				a.askCancelTransfer(task.ID)
			}
		}
	})
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
