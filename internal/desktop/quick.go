package desktop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/HugoSmits86/nativewebp"
	"github.com/amigoer/bucky/internal/core"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// pathTemplates are the image host layouts the panel offers.
var pathTemplates = []string{
	"images/{yyyy}/{mm}/{hash}.{ext}",
	"images/{yyyy}{mm}{dd}-{name}.{ext}",
	"screenshots/{name}.{ext}",
}

// openQuick opens the image host panel.
func (a *App) openQuick() {
	a.quick = quickState{open: true, format: a.prefs.Quick.Format}
	a.ensureQuickProfile()
}

// ensureQuickProfile picks the connection the panel uploads through.
func (a *App) ensureQuickProfile() {
	writable := a.writableProfiles()
	if len(writable) == 0 {
		return
	}
	chosen := a.prefs.Quick.ProfileID
	found := false
	for _, profile := range writable {
		if profile.ID == chosen {
			found = true
		}
	}
	if !found {
		a.prefs.Quick.ProfileID = writable[0].ID
		a.prefs.Quick.Bucket = writable[0].Bucket
		a.savePrefs()
	}
	a.ensureBuckets(a.prefs.Quick.ProfileID)
}

// writableProfiles are the connections the image host can use.
func (a *App) writableProfiles() []core.Profile {
	out := []core.Profile{}
	for _, profile := range a.profiles {
		if !profile.ReadOnly && profile.PublicURL != "" {
			out = append(out, profile)
		}
	}
	return out
}

// quickOverlay draws the image host panel.
func (a *App) quickOverlay(c *ui.Context) {
	state := &a.quick
	if !state.open {
		return
	}
	ui.Overlay(c, func() {
		backdrop := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Background(a.pal.scrim)
		backdrop.Children(func() {
			panel := ui.Column(c).Width(560).MaxHeightPercent(86).Radius(16).
				Background(a.pal.background).Border(1, a.pal.borderStrong).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.22))
			panel.Clip()
			panel.Children(func() {
				ui.Row(c).Height(52).PaddingX(18).Gap(10).AlignItems(ui.Center).
					BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
					ui.Icon(c, iconZap).FontSize(18).TextColor(a.pal.primary)
					ui.Text(c, a.t("quick.title")).FontSize(14).Bold().Grow(1)
					if a.iconAction(c, iconX, a.t("common.close")) {
						a.quick = quickState{}
					}
				})
				ui.Scroll(c).Grow(1).Children(func() {
					ui.Column(c).Gap(14).Padding(16, 18).Children(func() {
						a.quickImage(c)
						a.quickOptions(c)
					})
				})
				a.quickFooter(c)
			})
			if backdrop.OverlayShortcut(0, ui.KeyEscape) {
				a.quick = quickState{}
			}
		})
	})
}

// quickImage shows the image to upload, or invites one.
func (a *App) quickImage(c *ui.Context) {
	state := &a.quick
	drop := ui.Column(c).Height(200).Radius(12).Center().Gap(8).
		Background(a.pal.tile).Border(1, a.pal.borderStrong)
	if drop.FileDragOver() {
		drop.Border(2, a.pal.primary)
	}
	drop.Children(func() {
		switch {
		case state.busy:
			ui.Spinner(c).Size(22, 22)
			ui.Text(c, a.t("quick.processing")).FontSize(12).TextColor(a.pal.muted)
		case state.bitmap != nil:
			ui.Image(c, state.bitmap).Fit(ui.Contain).Grow(1).Padding(8)
		default:
			ui.Icon(c, iconImagePlus).FontSize(28).TextColor(a.pal.faint)
			ui.Text(c, a.t("quick.empty")).FontSize(12.5).TextColor(a.pal.muted)
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, a.t("quick.choose")).Clicked() {
					a.pickQuickImage()
				}
				if ui.Button(c, a.t("quick.clipboard")).Clicked() {
					a.pasteQuickImage()
				}
			})
		}
	})
	if files := drop.DroppedFiles(); len(files) > 0 {
		a.loadQuickFile(files[0])
	}
	if state.err != "" {
		ui.Text(c, state.err).FontSize(11.5).TextColor(a.pal.destructive)
	}
}

// quickOptions chooses where and how the image is uploaded.
func (a *App) quickOptions(c *ui.Context) {
	state := &a.quick
	writable := a.writableProfiles()
	if len(writable) == 0 {
		a.planNotice(c, a.pal.warningBg, a.pal.warningText, a.t("quick.noConnection"))
		return
	}
	profile := a.quickProfile()
	if profile == nil {
		return
	}
	a.settingsGroup(c, "", func() {
		a.fieldRow(c, a.t("quick.connection"), func() {
			options := make([]string, 0, len(writable))
			for _, item := range writable {
				options = append(options, item.Name)
			}
			current := profile.Name
			select_ := ui.Select(c, &current, options).Label(a.t("quick.connection")).Width(240)
			if select_.Changed() {
				for _, item := range writable {
					if item.Name == current {
						a.prefs.Quick.ProfileID = item.ID
						a.prefs.Quick.Bucket = item.Bucket
						a.savePrefs()
						a.ensureBuckets(item.ID)
						return
					}
				}
			}
		})
		a.fieldRow(c, a.t("quick.bucket"), func() {
			names := a.quickBuckets(profile)
			if len(names) == 0 {
				ui.Text(c, a.t("sidebar.noBuckets")).FontSize(12).TextColor(a.pal.muted)
				return
			}
			current := a.prefs.Quick.Bucket
			if current == "" {
				current = names[0]
			}
			select_ := ui.Select(c, &current, names).Label(a.t("quick.bucket")).Width(240)
			if select_.Changed() {
				a.prefs.Quick.Bucket = current
				a.savePrefs()
			}
		})
		a.fieldRow(c, a.t("quick.template"), func() {
			labels := append([]string{}, pathTemplates...)
			index := a.prefs.Quick.Template
			if index < 0 || index >= len(labels) {
				index = 0
			}
			current := labels[index]
			select_ := ui.Select(c, &current, labels).Label(a.t("quick.template")).Width(300)
			if select_.Changed() {
				for i, label := range labels {
					if label == current {
						a.prefs.Quick.Template = i
						a.savePrefs()
						return
					}
				}
			}
		})
		a.fieldRow(c, a.t("quick.compress"), func() {
			if ui.Switch(c, &a.prefs.Quick.Compress).Changed() {
				a.savePrefs()
				a.reencodeQuick()
			}
		})
		a.fieldRow(c, a.t("quick.webp"), func() {
			if ui.Switch(c, &a.prefs.Quick.WebP).Changed() {
				a.savePrefs()
				a.reencodeQuick()
			}
		})
		a.fieldRow(c, a.t("quick.copyAs"), func() {
			formats := []string{"markdown", "url"}
			labels := []string{"Markdown", "URL"}
			index := 0
			if a.prefs.Quick.Format == "url" {
				index = 1
			}
			seg := ui.Segmented(c, &index, labels...).Label(a.t("quick.copyAs")).Width(180)
			if seg.Changed() && index >= 0 && index < len(formats) {
				a.prefs.Quick.Format = formats[index]
				a.savePrefs()
			}
		})
	})
	_ = state
}

// quickProfile is the connection the panel uploads through.
func (a *App) quickProfile() *core.Profile {
	return a.profileOf(a.prefs.Quick.ProfileID)
}

// quickBuckets are the buckets the panel can upload to.
func (a *App) quickBuckets(profile *core.Profile) []string {
	if profile.Bucket != "" {
		return []string{profile.Bucket}
	}
	state := a.buckets[profile.ID]
	if state == nil {
		a.ensureBuckets(profile.ID)
		return nil
	}
	names := make([]string, 0, len(state.buckets))
	for _, bucket := range state.buckets {
		names = append(names, bucket.Name)
	}
	return names
}

// quickFooter uploads and copies the link.
func (a *App) quickFooter(c *ui.Context) {
	state := &a.quick
	ui.Column(c).Gap(8).Padding(12, 18).
		BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		if state.link != "" {
			ui.Text(c, state.link).FontSize(11).Font("monospace").
				TextColor(a.pal.muted).SingleLine().Selectable()
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if state.done {
				ui.Text(c, a.t("quick.done")).FontSize(12).TextColor(a.pal.successText).Grow(1)
				if ui.Button(c, a.t("quick.again")).Clicked() {
					a.quick = quickState{open: true, format: a.prefs.Quick.Format}
				}
				return
			}
			ui.Spacer(c)
			if state.conflict {
				if ui.Button(c, a.t("quick.rename")).Clicked() {
					a.uploadQuick(true)
				}
				if ui.PrimaryButton(c, a.t("quick.overwrite")).Clicked() {
					a.uploadQuick(false)
				}
				return
			}
			label := a.t("quick.upload")
			if state.busy {
				label = a.t("quick.uploading")
			}
			button := ui.PrimaryButton(c, label).
				Disabled(state.busy || len(state.data) == 0 || a.quickProfile() == nil)
			if button.Clicked() {
				a.uploadQuick(false)
			}
		})
	})
}

// pickQuickImage asks for an image file.
func (a *App) pickQuickImage() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.win, Title: a.t("quick.choose"),
			Filters: []mygo.FileFilter{{
				Name: "Images", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "bmp"},
			}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.loadQuickFile(paths[0])
	}()
}

// pasteQuickImage reads an image off the clipboard.
func (a *App) pasteQuickImage() {
	data := mygo.Clipboard.ReadImage()
	if len(data) == 0 {
		a.update(func() { a.quick.err = a.t("quick.empty") })
		return
	}
	a.applyQuickImage(data, "clipboard.png", "png")
}

// loadQuickFile reads a local image file into the panel.
func (a *App) loadQuickFile(filename string) {
	data, err := readFileLimited(filename, maxUploadData)
	if err != nil {
		a.update(func() { a.quick.err = a.errorText(err) })
		return
	}
	name := path.Base(filename)
	ext := extension(name)
	if ext == "" {
		ext = "png"
	}
	a.applyQuickImage(data, name, ext)
}

// applyQuickImage decodes an image and prepares it for upload.
func (a *App) applyQuickImage(data []byte, name, ext string) {
	a.update(func() {
		a.quick.err = ""
		a.quick.done = false
		a.quick.conflict = false
		a.quick.link = ""
		a.quick.source = ""
		a.quick.name = name
		a.quick.data = data
		a.quick.size = int64(len(data))
		a.quick.busy = true
	})
	go func() {
		encoded, width, height, err := encodeQuickImage(data, a.prefs.Quick.Compress, a.prefs.Quick.WebP)
		img, _, decodeErr := image.Decode(bytes.NewReader(data))
		if err == nil && decodeErr == nil {
			_ = width
			_ = height
		}
		a.update(func() {
			a.quick.busy = false
			if err != nil {
				a.quick.err = a.errorText(err)
				return
			}
			a.quick.data = encoded
			a.quick.size = int64(len(encoded))
			if img != nil {
				a.quick.bitmap = ui.NewBitmap(img)
				bounds := img.Bounds()
				a.quick.width, a.quick.height = bounds.Dx(), bounds.Dy()
			}
			a.quick.format = quickExtension(a.prefs.Quick.WebP)
		})
	}()
}

// reencodeQuick prepares the image again after a format change.
func (a *App) reencodeQuick() {
	if a.quick.source == "" {
		return
	}
	a.loadQuickFile(a.quick.source)
}

// quickExtension is the extension the panel uploads with.
func quickExtension(webp bool) string {
	if webp {
		return "webp"
	}
	return "jpg"
}

// encodeQuickImage re-encodes an image for the image host. Without
// compression the original bytes are kept as they are.
func encodeQuickImage(data []byte, compress, webp bool) ([]byte, int, int, error) {
	if !compress && !webp {
		return data, 0, 0, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, core.NewError("unsupported", "The image format cannot be read")
	}
	bounds := img.Bounds()
	var out bytes.Buffer
	if webp {
		// The pure Go encoder writes lossless WebP.
		if err := nativewebp.Encode(&out, img, &nativewebp.Options{CompressionLevel: nativewebp.BestCompression}); err != nil {
			return nil, 0, 0, core.NewError("unsupported", "The image cannot be converted to WebP")
		}
		return out.Bytes(), bounds.Dx(), bounds.Dy(), nil
	}
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 82}); err != nil {
		return nil, 0, 0, core.NewError("unsupported", "The image cannot be compressed")
	}
	return out.Bytes(), bounds.Dx(), bounds.Dy(), nil
}

// uploadQuick uploads the prepared image and copies its link.
func (a *App) uploadQuick(rename bool) {
	state := &a.quick
	profile := a.quickProfile()
	if profile == nil || len(state.data) == 0 {
		return
	}
	bucket := a.prefs.Quick.Bucket
	if bucket == "" {
		bucket = profile.Bucket
	}
	if bucket == "" {
		names := a.quickBuckets(profile)
		if len(names) == 0 {
			state.err = a.t("quick.noConnection")
			return
		}
		bucket = names[0]
	}
	suffix := ""
	if rename {
		suffix = "-" + time.Now().Format("150405")
	}
	key := renderTemplate(pathTemplates[a.prefs.Quick.Template], templateValues{
		date: time.Now(),
		hash: hashOf(state.data),
		name: state.name,
		ext:  quickExtension(a.prefs.Quick.WebP),
	}, suffix)
	ref := core.ObjectRef{ProfileID: profile.ID, Bucket: bucket, Key: key}
	data := state.data
	format := a.prefs.Quick.Format
	state.busy = true
	state.err = ""
	state.conflict = false

	go func() {
		result, err := a.uploadData(a.ctx, ref, data)
		if err != nil {
			a.update(func() {
				state.busy = false
				if core.AsError(err).Code == "conflict" {
					state.conflict = true
					state.err = a.t("quick.conflict")
					return
				}
				state.err = a.tf("quick.failed", "message", a.errorText(err))
			})
			return
		}
		if result.Plan != nil {
			a.update(func() {
				state.busy = false
				state.conflict = true
				state.err = a.t("quick.conflict")
				a.queuePlan(planRequest{operation: "upload", location: Location{
					ProfileID: profile.ID, Bucket: bucket, Prefix: parentPrefix(key),
				}, plan: result.Plan})
			})
			return
		}
		link, err := a.svc.Share(context.Background(), ref, 0, true)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.err = a.tf("quick.failed", "message", a.errorText(err))
				return
			}
			text := link.URL
			if format == "markdown" {
				text = markdownImage(state.name, link.URL)
			}
			state.link = link.URL
			state.done = true
			a.copyTextCurrent(text)
			a.loadTasks()
			if a.location != nil && a.location.ProfileID == profile.ID && a.location.Bucket == bucket {
				a.loadEntries()
			}
		})
	}()
}

// templateValues fill a path template.
type templateValues struct {
	date time.Time
	hash string
	name string
	ext  string
}

var unsafeName = regexp.MustCompile(`[\s/\\?#%*:|"<>]+`)
var repeatedDash = regexp.MustCompile(`-+`)
var repeatedSlash = regexp.MustCompile(`/{2,}`)

// sanitizeName makes a file name safe to use in an object key.
func sanitizeName(name string) string {
	base := name
	if dot := strings.LastIndex(base, "."); dot > 0 {
		base = base[:dot]
	}
	base = unsafeName.ReplaceAllString(strings.TrimSpace(base), "-")
	base = repeatedDash.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if len(base) > 80 {
		base = base[:80]
	}
	return base
}

// renderTemplate fills a path template; a suffix goes before the extension.
func renderTemplate(template string, values templateValues, suffix string) string {
	name := sanitizeName(values.name)
	if name == "" {
		name = "image"
	}
	key := template
	replacements := []string{
		"{yyyy}", values.date.Format("2006"),
		"{mm}", values.date.Format("01"),
		"{dd}", values.date.Format("02"),
		"{hh}", values.date.Format("15"),
		"{hash}", values.hash,
		"{name}", name,
		"{ext}", strings.ToLower(values.ext),
	}
	key = strings.NewReplacer(replacements...).Replace(key)
	if suffix != "" {
		slash := strings.LastIndex(key, "/")
		dot := strings.LastIndex(key, ".")
		if dot > slash {
			key = key[:dot] + suffix + key[dot:]
		} else {
			key += suffix
		}
	}
	key = repeatedSlash.ReplaceAllString(key, "/")
	return strings.TrimLeft(key, "/")
}

// hashOf is the short content hash a template can use.
func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:8]
}

// markdownImage writes an image reference for the clipboard.
func markdownImage(name, url string) string {
	alt := strings.NewReplacer("[", "", "]", "").Replace(name)
	return fmt.Sprintf("![%s](%s)", alt, url)
}

// readFileLimited reads a file, refusing one that is too large.
func readFileLimited(filename string, limit int64) ([]byte, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return nil, core.NewError("local_io", "Cannot read the selected file")
	}
	if info.Size() > limit {
		return nil, core.NewError("unsupported", "The selected file is too large")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, core.NewError("local_io", "Cannot read the selected file")
	}
	return data, nil
}

// copyTextCurrent puts text on the clipboard outside a build pass.
func (a *App) copyTextCurrent(text string) {
	mygo.Clipboard.WriteText(text)
	a.toastCurrent(a.t("toast.copied"))
}
