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
	"strconv"
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
	closeQuick := func() { a.quick = quickState{} }
	a.dialog(c, 520, false, false, closeQuick, func() {
		a.dialogHeader(c, iconZap, a.pal.primarySoft, a.pal.primaryText, a.t("quick.title"), "", func() {
			if platformIsMac() {
				ui.Text(c, "⌥⌘U").FontSize(11).TextColor(a.pal.faint)
			}
		}, closeQuick)
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			a.quickImage(c)
			a.quickOptions(c)
			a.quickStatus(c)
		})
		a.quickFooter(c)
	})
	if !state.busy && c.Shortcut(ui.Cmd, ui.KeyV) {
		a.pasteQuickImage()
	}
}

// quickImage shows the image to upload, or invites one.
func (a *App) quickImage(c *ui.Context) {
	state := &a.quick
	row := ui.Row(c).Gap(14).Padding(0, 16, 14, 16)
	dragging := row.FileDragOver()
	row.Children(func() {
		thumb := ui.Box(c).Size(150, 108).Radius(10).Clip().Center().Shrink(0).Background(a.pal.tile)
		if dragging {
			thumb.Border(2, a.pal.primary)
		} else {
			thumb.Border(1, a.pal.border)
		}
		if state.bitmap == nil {
			thumb.Stripes(a.pal.stripe, 6, 6, 135)
		}
		thumb.Children(func() {
			if state.bitmap != nil {
				ui.Image(c, state.bitmap).Fit(ui.Cover).Absolute().Left(0).Top(0).Right(0).Bottom(0)
			}
			switch {
			case state.busy:
				ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
					Background(ui.RGBA(0, 0, 0, 0.35)).Children(func() {
					ui.Spinner(c).Size(18, 18).Label(a.t("quick.processing"))
				})
			case state.done:
				ui.Box(c).Absolute().Right(7).Top(7).Size(22, 22).Radius(11).Center().
					Background(a.pal.success).Children(func() {
					ui.Icon(c, iconCheck).FontSize(12).TextColor(ui.Hex("#ffffff"))
				})
			case state.bitmap == nil:
				ui.Icon(c, iconImagePlus).FontSize(26).TextColor(a.pal.faint)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Padding(2, 0, 0, 0).Children(func() {
			if len(state.data) > 0 {
				ui.Text(c, state.name).FontSize(13).FontWeight(600).SingleLine()
				detail := formatBytes(state.size)
				if state.width > 0 {
					detail = strconv.Itoa(state.width) + " × " + strconv.Itoa(state.height) + " · " + detail
				}
				ui.Text(c, detail).FontSize(12).FontFeatures("tnum").TextColor(a.pal.muted)
			} else {
				ui.Text(c, a.t("quick.empty")).FontSize(13).FontWeight(600)
			}
			ui.Row(c).Gap(6).Margin(6, 0, 0, 0).Children(func() {
				if a.secondaryButton(c, nil, a.t("quick.choose")).Height(26).Clicked() {
					a.pickQuickImage()
				}
				if a.secondaryButton(c, nil, a.t("quick.clipboard")).Height(26).Clicked() {
					a.pasteQuickImage()
				}
			})
			ui.Spacer(c)
			if platformIsMac() {
				ui.Text(c, a.t("quick.paste")).FontSize(11.5).TextColor(a.pal.faint)
			}
		})
	})
	if files := row.DroppedFiles(); len(files) > 0 {
		a.loadQuickFile(files[0])
	}
}

// quickOptions chooses where and how the image is uploaded.
func (a *App) quickOptions(c *ui.Context) {
	writable := a.writableProfiles()
	ui.Column(c).Gap(10).Padding(14, 16).BorderWidth(1, 0, 0, 0).BorderColor(a.pal.border).Children(func() {
		if len(writable) == 0 {
			a.notice(c, a.pal.warningBg, a.pal.warningText, a.t("quick.noConnection"))
			return
		}
		profile := a.quickProfile()
		if profile == nil {
			return
		}
		ui.Row(c).Gap(10).Children(func() {
			a.quickField(c, a.t("quick.connection"), func() {
				options := make([]string, 0, len(writable))
				for _, item := range writable {
					options = append(options, item.Name)
				}
				current := profile.Name
				select_ := ui.Select(c, &current, options).Label(a.t("quick.connection")).FillWidth()
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
			a.quickField(c, a.t("quick.bucket"), func() {
				names := a.quickBuckets(profile)
				if len(names) == 0 {
					ui.Text(c, a.t("sidebar.noBuckets")).FontSize(12).TextColor(a.pal.muted).Height(30)
					return
				}
				current := a.prefs.Quick.Bucket
				if current == "" {
					current = names[0]
				}
				select_ := ui.Select(c, &current, names).Label(a.t("quick.bucket")).FillWidth()
				if select_.Changed() {
					a.prefs.Quick.Bucket = current
					a.savePrefs()
				}
			})
		})
		a.quickField(c, a.t("quick.template"), func() {
			labels := append([]string{}, pathTemplates...)
			index := a.prefs.Quick.Template
			if index < 0 || index >= len(labels) {
				index = 0
			}
			current := labels[index]
			select_ := ui.Select(c, &current, labels).Label(a.t("quick.template")).FillWidth()
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
		ui.Text(c, "→ "+a.quickKey("")).FontSize(11.5).Font("monospace").TextColor(a.pal.muted).SingleLine()
		ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
			a.quickSwitch(c, a.t("quick.compress"), &a.prefs.Quick.Compress)
			a.quickSwitch(c, a.t("quick.webp"), &a.prefs.Quick.WebP)
			ui.Spacer(c)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Text(c, a.t("quick.copyAs")).FontSize(11.5).TextColor(a.pal.muted).SingleLine()
				formats := []string{"markdown", "url"}
				index := 0
				if a.prefs.Quick.Format == "url" {
					index = 1
				}
				if a.segments(c, &index, a.t("quick.copyAs"), []string{"Markdown", "URL"}) &&
					index >= 0 && index < len(formats) {
					a.prefs.Quick.Format = formats[index]
					a.savePrefs()
				}
			})
		})
	})
}

// quickField is a labeled control of the quick upload options.
func (a *App) quickField(c *ui.Context, label string, control func()) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(5).Children(func() {
		ui.Text(c, label).FontSize(11.5).TextColor(a.pal.muted)
		control()
	})
}

// quickSwitch is a labeled switch that prepares the image again.
func (a *App) quickSwitch(c *ui.Context, label string, value *bool) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		if ui.Switch(c, value).Label(label).Changed() {
			a.savePrefs()
			a.reencodeQuick()
		}
		ui.Text(c, label).FontSize(12.5).SingleLine()
	})
}

// quickKey is the key the image would be uploaded as. Before an image is
// chosen it shows the template's placeholders.
func (a *App) quickKey(suffix string) string {
	state := &a.quick
	template := pathTemplates[0]
	if index := a.prefs.Quick.Template; index >= 0 && index < len(pathTemplates) {
		template = pathTemplates[index]
	}
	if len(state.data) == 0 {
		return template
	}
	return renderTemplate(template, templateValues{
		date: time.Now(),
		hash: state.hash,
		name: state.name,
		ext:  quickExtension(a.prefs.Quick.WebP),
	}, suffix)
}

// quickStatus reports the upload: in progress, done, refused or failed.
func (a *App) quickStatus(c *ui.Context) {
	state := &a.quick
	if state.err == "" && !state.done && (!state.busy || len(state.data) == 0) {
		return
	}
	ui.Column(c).Gap(8).Padding(0, 16, 14, 16).Children(func() {
		switch {
		case state.conflict:
			ui.Column(c).Gap(8).Padding(10, 12).Radius(10).Background(a.pal.warningBg).Children(func() {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconAlertTriangle).FontSize(13).TextColor(a.pal.warningText)
					ui.Text(c, a.t("quick.conflict")).FontSize(12.5).FontWeight(600).TextColor(a.pal.warningText)
				})
				ui.Text(c, a.quickKey("")).FontSize(11.5).Font("monospace").TextColor(a.pal.warningText).SingleLine()
				ui.Row(c).Gap(6).Children(func() {
					if a.secondaryButton(c, nil, a.t("quick.rename")).Height(26).Clicked() {
						a.uploadQuick(true)
					}
					if a.secondaryButton(c, nil, a.t("quick.overwrite")).Height(26).Clicked() {
						a.uploadQuick(false)
					}
				})
			})
		case state.err != "":
			a.notice(c, a.pal.destructiveBg, a.pal.destructive, state.err)
		case state.done:
			ui.Column(c).Gap(7).Padding(10, 12).Radius(10).Background(a.pal.successBg).Children(func() {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconCheck).FontSize(13).TextColor(a.pal.successText)
					ui.Text(c, a.t("quick.done")).FontSize(12.5).FontWeight(600).TextColor(a.pal.successText)
				})
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, state.link).FontSize(11.5).Font("monospace").SingleLine().Selectable().
						Grow(1).MinWidth(0)
					if a.secondaryButton(c, iconCopy, a.t("common.copy")).Height(26).Clicked() {
						a.copyText(c, state.link)
					}
				})
			})
		default:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(11, 11)
				ui.Text(c, a.t("quick.uploading")).FontSize(12).TextColor(a.pal.muted)
			})
		}
	})
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

// quickFooter closes the panel or uploads and copies the link.
func (a *App) quickFooter(c *ui.Context) {
	state := &a.quick
	a.dialogFooter(c, func() {
		ui.Spacer(c)
		if a.secondaryButton(c, nil, a.t("common.close")).Height(30).Clicked() {
			a.quick = quickState{}
			return
		}
		if state.done {
			if a.primaryAction(c, iconUpload, a.t("quick.again"), false).Height(30).Clicked() {
				a.quick = quickState{open: true, format: a.prefs.Quick.Format}
			}
			return
		}
		label := a.t("quick.upload")
		if state.busy && len(state.data) > 0 {
			label = a.t("quick.uploading")
		}
		disabled := state.busy || state.conflict || len(state.data) == 0 || a.quickProfile() == nil
		if a.primaryAction(c, iconUpload, label, disabled).Height(30).Clicked() && !disabled {
			a.uploadQuick(false)
		}
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
			a.quick.hash = hashOf(encoded)
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
	key := a.quickKey(suffix)
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
