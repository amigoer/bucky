package desktop

import (
	"bytes"
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"sort"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// maxPreviewImage caps what the preview downloads to draw inline.
const maxPreviewImage = 32 << 20

// openPreview shows the Quick Look panel for an object.
func (a *App) openPreview(key string) {
	if a.location == nil {
		return
	}
	state := &a.preview
	state.open = true
	state.key = key
	state.list = append([]entry{}, a.browse.shown...)
	state.info = core.ObjectInfo{}
	state.text = core.TextPreview{}
	state.err = nil
	state.busy = true
	state.link = core.ShareLink{}
	state.linkAt = 0

	location := *a.location
	ref := core.ObjectRef{ProfileID: location.ProfileID, Bucket: location.Bucket, Key: key}
	go func() {
		info, err := a.svc.Stat(a.ctx, ref)
		if err != nil {
			a.update(func() {
				if a.preview.key != key {
					return
				}
				a.preview.busy = false
				a.preview.err = err
			})
			return
		}
		mode := previewMode(kindOf(info.Key, info.ContentType, false))
		var text core.TextPreview
		if mode == "text" {
			text, err = a.svc.ReadText(a.ctx, ref, 1<<20)
		}
		var link core.ShareLink
		if mode == "image" {
			link, err = a.svc.Share(a.ctx, ref, 3600, false)
		}
		a.update(func() {
			if a.preview.key != key {
				return
			}
			a.preview.busy = false
			a.preview.info = info
			a.preview.text = text
			a.preview.link = link
			a.preview.linkAt = time.Now().Unix()
			if err != nil {
				a.preview.err = err
			}
		})
	}()
}

// closePreview hides the panel.
func (a *App) closePreview() {
	a.preview = previewState{}
}

// previewOverlay draws the Quick Look panel over the window: the preview,
// and the object's facts beside it.
func (a *App) previewOverlay(c *ui.Context) {
	state := &a.preview
	if !state.open {
		return
	}
	overlay := a.dialog(c, 860, false, false, a.closePreview, func() {
		ui.Row(c).Height(42).Padding(0, 10, 0, 12).Gap(10).AlignItems(ui.Center).Shrink(0).
			BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
			a.ghostButton(c, iconX, a.t("common.close"), 28, false, a.closePreview)
			ui.Text(c, basename(state.key)).FontSize(13).FontWeight(600).SingleLine().
				TextAlign(ui.Center).Grow(1).MinWidth(0)
			ui.Text(c, a.t("quicklook.hint")).FontSize(11.5).TextColor(a.pal.faint).SingleLine()
		})
		ui.Row(c).Height(420).Shrink(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Clip().Children(func() { a.previewBody(c) })
			a.previewMeta(c)
		})
	})
	if overlay.OverlayShortcut(0, ui.KeySpace) {
		a.closePreview()
	}
	if overlay.OverlayShortcut(0, ui.KeyLeft) {
		a.stepPreview(-1)
	}
	if overlay.OverlayShortcut(0, ui.KeyRight) {
		a.stepPreview(1)
	}
}

// stepPreview moves to the previous or next object of the folder.
func (a *App) stepPreview(delta int) {
	state := &a.preview
	index := -1
	for i, row := range state.list {
		if row.key == state.key {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	for next := index + delta; next >= 0 && next < len(state.list); next += delta {
		if state.list[next].folder {
			continue
		}
		a.openPreview(state.list[next].key)
		return
	}
}

// previewBody draws what the preview can show of the object.
func (a *App) previewBody(c *ui.Context) {
	state := &a.preview
	switch {
	case state.busy:
		a.previewLoading(c)
		return
	case state.err != nil:
		a.previewNotice(c, a.t("quicklook.failed"), a.errorText(state.err), true)
		return
	}

	kind := kindOf(state.info.Key, state.info.ContentType, false)
	switch previewMode(kind) {
	case "image":
		a.previewImage(c)
	case "text":
		a.previewText(c)
	default:
		a.previewNotice(c, a.t("quicklook.unsupported"), a.t("quicklook.unsupportedDesc"), false)
	}
}

func (a *App) previewLoading(c *ui.Context) {
	ui.Row(c).Fill().Center().Gap(8).Children(func() {
		ui.Spinner(c).Size(12, 12).Label(a.t("quicklook.loading"))
		ui.Text(c, a.t("quicklook.loading")).FontSize(12.5).TextColor(a.pal.muted)
	})
}

// previewImage fetches and draws the object's bitmap over a striped tile.
func (a *App) previewImage(c *ui.Context) {
	state := &a.preview
	if state.link.URL == "" {
		a.previewNotice(c, a.t("quicklook.unsupported"), a.t("quicklook.unsupportedDesc"), false)
		return
	}
	if state.bitmap == nil && !state.fetching {
		state.fetching = true
		go a.fetchPreviewImage(state.key, state.link)
	}
	ui.Column(c).Fill().Padding(16).Center().Background(a.pal.tile).
		Stripes(a.pal.stripe, 8, 8, 135).Children(func() {
		if state.bitmap == nil {
			ui.Spinner(c).Size(16, 16).Label(a.t("quicklook.loading"))
			return
		}
		ui.Image(c, state.bitmap).Fit(ui.Contain).Grow(1).FillWidth()
	})
}

// fetchPreviewImage downloads the object and decodes it for the preview.
func (a *App) fetchPreviewImage(key string, link core.ShareLink) {
	img, err := a.readImage(a.ctx, link)
	a.update(func() {
		if a.preview.key != key {
			return
		}
		a.preview.fetching = false
		if err != nil {
			a.preview.err = err
			return
		}
		a.preview.bitmap = ui.NewBitmap(img)
	})
}

// readImage reads an image over its presigned link.
func (a *App) readImage(ctx context.Context, link core.ShareLink) (image.Image, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, link.URL, nil)
	if err != nil {
		return nil, core.NewError("protocol", "Cannot build the preview request")
	}
	for name, value := range link.Headers {
		request.Header.Set(name, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, core.NewError("network", "Cannot load the preview")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, core.NewError("protocol", "Cannot load the preview")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxPreviewImage+1))
	if err != nil {
		return nil, core.NewError("network", "Cannot load the preview")
	}
	if int64(len(data)) > maxPreviewImage {
		return nil, core.NewError("unsupported", "The image is too large to preview")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, core.NewError("unsupported", "The image format cannot be previewed")
	}
	return img, nil
}

// previewText shows the first part of a text object.
func (a *App) previewText(c *ui.Context) {
	state := &a.preview
	ui.Column(c).Fill().Background(a.pal.popover).Children(func() {
		if state.text.Truncated {
			ui.Row(c).Gap(6).Padding(8, 14).AlignItems(ui.Center).Background(a.pal.warningBg).Children(func() {
				ui.Icon(c, iconAlertTriangle).FontSize(12).TextColor(a.pal.warningText)
				ui.Text(c, a.tf("quicklook.truncated", "size", formatBytes(state.info.Size))).
					FontSize(12).TextColor(a.pal.warningText)
			})
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(14, 16).Children(func() {
			ui.Text(c, state.text.Text).FontSize(12).LineHeight(1.65).Font("monospace").Selectable()
		})
	})
}

// previewNotice explains why the preview shows no content.
func (a *App) previewNotice(c *ui.Context, title, description string, retry bool) {
	ui.Column(c).Fill().Center().Padding(20).Children(func() {
		ui.Column(c).MaxWidth(340).Gap(8).AlignItems(ui.Center).Children(func() {
			tile, ink := a.pal.mutedFill, a.pal.muted
			if retry {
				tile, ink = a.pal.destructiveBg, a.pal.destructive
			}
			ui.Box(c).Size(48, 48).Radius(14).Center().Background(tile).Children(func() {
				ui.Icon(c, iconAlertTriangle).FontSize(18).TextColor(ink)
			})
			ui.Text(c, title).FontSize(14).FontWeight(600).TextAlign(ui.Center)
			ui.Text(c, description).FontSize(12.5).LineHeight(1.6).TextColor(a.pal.muted).TextAlign(ui.Center)
			ui.Row(c).Gap(8).Margin(6, 0, 0, 0).Children(func() {
				if retry {
					if a.secondaryButton(c, iconRefresh, a.t("quicklook.regenerate")).Height(26).Clicked() {
						a.openPreview(a.preview.key)
					}
				}
				if a.secondaryButton(c, iconDownload, a.t("common.download")).Height(26).Clicked() {
					a.download([]entry{{key: a.preview.key, name: basename(a.preview.key)}})
				}
			})
		})
	})
}

// previewMeta lists the object's facts and metadata beside the preview.
func (a *App) previewMeta(c *ui.Context) {
	state := &a.preview
	info := state.info
	kind := kindOf(info.Key, info.ContentType, false)
	ui.Scroll(c).Width(260).Shrink(0).BorderWidth(0, 0, 0, 1).BorderColor(a.pal.border).Children(func() {
		ui.Column(c).Gap(12).Padding(16).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, basename(state.key)).FontSize(14).FontWeight(650)
				sub := a.t("kinds." + kind)
				if info.Key != "" {
					sub += " · " + formatBytes(info.Size)
				}
				ui.Text(c, sub).FontSize(12).TextColor(a.pal.muted)
			})
			ui.Row(c).Gap(6).Children(func() {
				if a.secondaryButton(c, iconDownload, a.t("common.download")).Height(26).Grow(1).Clicked() {
					a.download([]entry{{key: state.key, name: basename(state.key)}})
				}
				if a.secondaryButton(c, iconLink, a.t("common.share")).Height(26).Grow(1).Clicked() {
					key := state.key
					a.closePreview()
					a.openShare([]entry{{key: key, name: basename(key)}})
				}
			})
			if info.Key == "" {
				return
			}
			ui.Column(c).Children(func() {
				a.metaField(c, a.t("quicklook.key"), info.Key, true)
				a.metaField(c, a.t("quicklook.size"), formatBytes(info.Size), false)
				a.metaField(c, a.t("quicklook.modified"), formatDate(info.LastModified), false)
				a.metaField(c, a.t("quicklook.contentType"), displayType(info.Key, info.ContentType), false)
				if info.ETag != "" {
					a.metaField(c, a.t("quicklook.etag"), info.ETag, true)
				}
			})
			ui.Column(c).Gap(5).Children(func() {
				ui.Text(c, a.t("quicklook.metadata")).FontSize(11).TextColor(a.pal.faint)
				if len(info.Metadata) == 0 {
					ui.Text(c, a.t("common.none")).FontSize(12).TextColor(a.pal.muted)
				}
				keys := make([]string, 0, len(info.Metadata))
				for name := range info.Metadata {
					keys = append(keys, name)
				}
				sort.Strings(keys)
				for _, name := range keys {
					ui.Column(c).Padding(6, 8).Radius(7).Background(a.pal.sidebar).Children(func() {
						ui.Text(c, name).FontSize(11).Font("monospace").TextColor(a.pal.muted)
						ui.Text(c, info.Metadata[name]).FontSize(11).Font("monospace").Selectable()
					})
				}
			})
			ui.Text(c, a.t("quicklook.note")).FontSize(11).LineHeight(1.5).TextColor(a.pal.faint)
		})
	})
}

// metaField is one labeled fact of the object.
func (a *App) metaField(c *ui.Context, label, value string, mono bool) {
	ui.Column(c).Gap(1).Padding(7, 0).BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(a.pal.faint)
		text := ui.Text(c, value).FontSize(12).Selectable()
		if mono {
			text.Font("monospace")
		}
	})
}

// openExternally opens a link in the default browser.
func (a *App) openExternally(url string) {
	if url == "" {
		return
	}
	if err := mygo.Shell.OpenExternal(url); err != nil {
		a.toastCurrent(a.t("errors.browser"))
	}
}
