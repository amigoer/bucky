package desktop

import (
	"bytes"
	"context"
	"fmt"
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

// previewOverlay draws the Quick Look panel over the window.
func (a *App) previewOverlay(c *ui.Context) {
	state := &a.preview
	if !state.open {
		return
	}
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Background(a.pal.scrim)
		backdrop.Children(func() {
			panel := ui.Column(c).Width(720).MaxHeightPercent(88).Radius(16).
				Background(a.pal.background).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.22))
			panel.Clip()
			panel.Children(func() {
				a.previewHeader(c)
				ui.Column(c).Grow(1).Children(func() { a.previewBody(c) })
				a.previewMeta(c)
			})
			if backdrop.PressedOutside() || backdrop.OverlayShortcut(0, ui.KeyEscape) {
				a.closePreview()
			}
		})
	})
}

// previewHeader names the object and offers the panel's actions.
func (a *App) previewHeader(c *ui.Context) {
	state := &a.preview
	ui.Row(c).Height(46).PaddingX(16).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
		ui.Icon(c, iconOf(kindOf(state.info.Key, state.info.ContentType, false))).
			FontSize(18).TextColor(a.pal.kindPaletteOf(kindOf(state.info.Key, state.info.ContentType, false)).foreground)
		ui.Text(c, basename(state.key)).FontSize(13.5).Bold().SingleLine().Grow(1)
		if a.iconAction(c, iconChevronLeft, a.t("browser.back")) {
			a.stepPreview(-1)
		}
		if a.iconAction(c, iconChevronRight, a.t("browser.forward")) {
			a.stepPreview(1)
		}
		if a.iconAction(c, iconX, a.t("common.close")) {
			a.closePreview()
		}
	})
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
		ui.Column(c).Height(320).Center().Gap(10).Children(func() {
			ui.Spinner(c).Label(a.t("quicklook.loading"))
			ui.Text(c, a.t("quicklook.loading")).FontSize(12.5).TextColor(a.pal.muted)
		})
		return
	case state.err != nil:
		a.previewNotice(c, iconCircleAlert, a.t("quicklook.failed"), a.errorText(state.err), true)
		return
	}

	kind := kindOf(state.info.Key, state.info.ContentType, false)
	switch previewMode(kind) {
	case "image":
		a.previewImage(c, kind)
	case "text":
		a.previewText(c)
	default:
		a.previewNotice(c, iconOf(kind), a.t("quicklook.unsupported"),
			a.t("quicklook.unsupportedDesc"), false)
	}
}

// previewImage fetches and draws the object's bitmap.
func (a *App) previewImage(c *ui.Context, kind string) {
	state := &a.preview
	if state.link.URL == "" {
		a.previewNotice(c, iconImage, a.t("quicklook.unsupported"),
			a.t("quicklook.unsupportedDesc"), false)
		return
	}
	if state.bitmap == nil {
		if !state.fetching {
			state.fetching = true
			go a.fetchPreviewImage(state.key, state.link)
		}
		ui.Column(c).Height(320).Center().Children(func() { ui.Spinner(c) })
		return
	}
	ui.Column(c).Height(320).Padding(16).Center().Background(a.pal.tile).Children(func() {
		ui.Image(c, state.bitmap).Fit(ui.Contain).Grow(1)
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
	ui.Column(c).Grow(1).Children(func() {
		if state.text.Truncated {
			ui.Row(c).PaddingX(16).PaddingY(6).Children(func() {
				ui.Text(c, a.tf("quicklook.truncated", "size", formatBytes(state.info.Size))).
					FontSize(11.5).TextColor(a.pal.muted)
			})
		}
		ui.Scroll(c).Height(320).Padding(16).Children(func() {
			ui.Text(c, state.text.Text).FontSize(12).Font("monospace").Selectable()
		})
	})
}

// previewNotice explains why the preview shows no content.
func (a *App) previewNotice(c *ui.Context, glyph *ui.SVG, title, description string, retry bool) {
	ui.Column(c).Height(320).Center().Padding(24).Children(func() {
		ui.Column(c).MaxWidth(360).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, glyph).FontSize(30).TextColor(a.pal.faint)
			ui.Text(c, title).FontSize(13.5).Bold().TextAlign(ui.Center)
			ui.Text(c, description).FontSize(12).TextColor(a.pal.muted).TextAlign(ui.Center)
			ui.Row(c).Gap(8).Children(func() {
				if retry {
					if ui.Button(c, a.t("quicklook.regenerate")).Clicked() {
						a.openPreview(a.preview.key)
					}
				}
				if ui.Button(c, a.t("common.download")).Clicked() {
					a.download([]entry{{key: a.preview.key, name: basename(a.preview.key)}})
				}
			})
		})
	})
}

// previewMeta lists the object's metadata under the preview.
func (a *App) previewMeta(c *ui.Context) {
	state := &a.preview
	if state.info.Key == "" {
		return
	}
	ui.Column(c).Gap(4).Padding(12, 16, 14, 16).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).
		Background(a.pal.sidebar).Children(func() {
		ui.Row(c).Gap(12).Children(func() {
			a.metaCell(c, a.t("quicklook.key"), state.info.Key, true)
			a.metaCell(c, a.t("quicklook.size"), formatBytes(state.info.Size), false)
			a.metaCell(c, a.t("quicklook.modified"), formatDate(state.info.LastModified), false)
			a.metaCell(c, a.t("quicklook.contentType"),
				displayType(state.info.Key, state.info.ContentType), false)
		})
		if state.info.ETag != "" {
			a.metaCell(c, a.t("quicklook.etag"), state.info.ETag, true)
		}
		if len(state.info.Metadata) > 0 {
			keys := make([]string, 0, len(state.info.Metadata))
			for name := range state.info.Metadata {
				keys = append(keys, name)
			}
			sort.Strings(keys)
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, a.t("quicklook.metadata")).FontSize(10.5).TextColor(a.pal.faint)
				for _, name := range keys {
					ui.Text(c, fmt.Sprintf("%s: %s", name, state.info.Metadata[name])).
						FontSize(11.5).TextColor(a.pal.muted).SingleLine()
				}
			})
		}
		ui.Text(c, a.t("quicklook.note")).FontSize(10.5).TextColor(a.pal.faint)
	})
}

// metaCell is one labelled value of the metadata block.
func (a *App) metaCell(c *ui.Context, label, value string, grow bool) {
	cell := ui.Column(c).Gap(2)
	if grow {
		cell.Grow(1)
	}
	cell.Children(func() {
		ui.Text(c, label).FontSize(10.5).TextColor(a.pal.faint)
		ui.Text(c, value).FontSize(11.5).SingleLine().TextColor(a.pal.muted)
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
