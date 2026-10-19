package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
)

const (
	maxUploadData = 64 << 20
	stagedMaxAge  = time.Hour
)

type DownloadResult struct {
	Key   string             `json:"key"`
	Task  *core.TransferTask `json:"task,omitempty"`
	Error *core.Error        `json:"error,omitempty"`
}

// DownloadTo queues one download per object below dir. Keys keep their path
// below base; without a base only the object name is used. Folder markers
// only create directories.
func (d *DesktopService) DownloadTo(ctx context.Context, dir string, sources []core.ObjectRef, base string) ([]DownloadResult, error) {
	if len(sources) == 0 || len(sources) > 1000 {
		return nil, core.Invalid("Select between 1 and 1000 objects to download")
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, core.Invalid("Select an existing download directory")
	}
	results := make([]DownloadResult, 0, len(sources))
	for _, ref := range sources {
		result := DownloadResult{Key: ref.Key}
		rel, ok := localTarget(ref.Key, base)
		switch {
		case !ok:
			result.Error = core.Invalid("The object key cannot be used as a local file name")
		case strings.HasSuffix(ref.Key, "/"):
			if err := os.MkdirAll(filepath.Join(dir, rel), 0o755); err != nil {
				result.Error = core.NewError("local_io", "Cannot create a download directory")
			}
		default:
			target := filepath.Join(dir, rel)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				result.Error = core.NewError("local_io", "Cannot create a download directory")
				break
			}
			started, err := d.backend.StartTransfer(ctx, core.TransferRequest{Direction: "download", Object: ref, LocalPath: target})
			if err != nil {
				result.Error = core.AsError(err)
			} else {
				result.Task = started.Task
			}
		}
		results = append(results, result)
	}
	return results, nil
}

// localTarget maps an object key to a relative local path that cannot escape
// the download directory.
func localTarget(key, base string) (string, bool) {
	rel := strings.TrimSuffix(key, "/")
	if base != "" && strings.HasPrefix(key, base) && key != base {
		rel = strings.TrimSuffix(strings.TrimPrefix(key, base), "/")
	} else {
		rel = path.Base(rel)
	}
	if rel == "" || rel == "." || strings.ContainsAny(rel, "\\\x00") {
		return "", false
	}
	local := filepath.FromSlash(rel)
	if !filepath.IsLocal(local) {
		return "", false
	}
	return local, true
}

// UploadData stages in-memory content, such as a pasted image, and uploads it
// through the transfer engine so overwrites still require a plan.
func (d *DesktopService) UploadData(ctx context.Context, ref core.ObjectRef, data []byte) (service.TransferResult, error) {
	if len(data) == 0 || len(data) > maxUploadData {
		return service.TransferResult{}, core.Invalid("Upload data must be between 1 byte and 64 MiB")
	}
	dir := d.stagedDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return service.TransferResult{}, core.NewError("local_io", "Cannot create the upload staging directory")
	}
	filename := filepath.Join(dir, stagedName(ref.Key))
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return service.TransferResult{}, core.NewError("local_io", "Cannot stage the upload data")
	}
	result, err := d.backend.StartTransfer(ctx, core.TransferRequest{Direction: "upload", Object: ref, LocalPath: filename})
	if err != nil {
		_ = os.Remove(filename)
	}
	return result, err
}

func (d *DesktopService) stagedDir() string { return filepath.Join(d.backend.DB.Dir, "staged") }

// stagedName keeps the key's extension so the upload gets a matching content type.
func stagedName(key string) string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("secure random source unavailable")
	}
	ext := strings.ToLower(path.Ext(key))
	if len(ext) > 16 || strings.ContainsAny(ext, "/\\\x00") {
		ext = ""
	}
	return hex.EncodeToString(raw[:]) + ext
}

// releaseStaged removes staged content once its upload can no longer resume.
func (d *DesktopService) releaseStaged(task core.TransferTask) {
	if task.Direction != "upload" || (task.Status != "completed" && task.Status != "cancelled") {
		return
	}
	if filepath.Dir(task.LocalPath) == d.stagedDir() {
		_ = os.Remove(task.LocalPath)
	}
}

// sweepStaged removes staged files that no unfinished upload needs, such as
// those left by overwrite plans that were never executed.
func (d *DesktopService) sweepStaged(ctx context.Context) {
	entries, err := os.ReadDir(d.stagedDir())
	if err != nil {
		return
	}
	tasks, err := d.backend.Tasks(ctx)
	if err != nil {
		return
	}
	keep := map[string]bool{}
	for _, t := range tasks {
		if t.Direction == "upload" && t.Status != "completed" && t.Status != "cancelled" {
			keep[t.LocalPath] = true
		}
	}
	for _, entry := range entries {
		filename := filepath.Join(d.stagedDir(), entry.Name())
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || keep[filename] || time.Since(info.ModTime()) < stagedMaxAge {
			continue
		}
		_ = os.Remove(filename)
	}
}
