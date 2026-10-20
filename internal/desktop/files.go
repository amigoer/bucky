package desktop

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
	maxPlanItems  = 1000
	stagedMaxAge  = time.Hour
)

// downloadResult is the outcome of queueing one download.
type downloadResult struct {
	key  string
	task *core.TransferTask
	err  *core.Error
}

// downloadTo queues one download per object below dir. Keys keep their path
// below base; without a base only the object name is used. Folder markers
// only create directories.
func (a *App) downloadTo(ctx context.Context, dir string, sources []core.ObjectRef, base string) ([]downloadResult, error) {
	if len(sources) == 0 || len(sources) > maxPlanItems {
		return nil, core.Invalid("Select between 1 and 1000 objects to download")
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, core.Invalid("Select an existing download directory")
	}
	results := make([]downloadResult, 0, len(sources))
	for _, ref := range sources {
		result := downloadResult{key: ref.Key}
		rel, ok := localTarget(ref.Key, base)
		switch {
		case !ok:
			result.err = core.Invalid("The object key cannot be used as a local file name")
		case strings.HasSuffix(ref.Key, "/"):
			if err := os.MkdirAll(filepath.Join(dir, rel), 0o755); err != nil {
				result.err = core.NewError("local_io", "Cannot create a download directory")
			}
		default:
			target := filepath.Join(dir, rel)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				result.err = core.NewError("local_io", "Cannot create a download directory")
				break
			}
			started, err := a.svc.StartTransfer(ctx, core.TransferRequest{
				Direction: "download", Object: ref, LocalPath: target,
			})
			if err != nil {
				result.err = core.AsError(err)
			} else {
				result.task = started.Task
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

// uploadPaths uploads local files and the contents of local folders to a
// prefix, queueing what can be queued and returning the plans an overwrite
// needs reviewed.
func (a *App) uploadPaths(ctx context.Context, location Location, paths []string) ([]service.TransferResult, error) {
	results := []service.TransferResult{}
	normalized := strings.Trim(location.Prefix, "/")
	if normalized != "" {
		normalized += "/"
	}
	seen := map[string]bool{}
	for _, filename := range paths {
		info, err := os.Lstat(filename)
		if err != nil {
			return results, core.Invalid("Cannot inspect a selected path")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		base := filepath.Dir(filename)
		err = filepath.WalkDir(filename, func(name string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return core.NewError("local_io", "Cannot read a selected directory")
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			rel, err := filepath.Rel(base, name)
			if err != nil {
				return core.Invalid("Cannot determine the upload path")
			}
			key := normalized + filepath.ToSlash(rel)
			if seen[key] {
				return nil
			}
			seen[key] = true
			result, err := a.svc.StartTransfer(ctx, core.TransferRequest{
				Direction: "upload", LocalPath: name,
				Object: core.ObjectRef{ProfileID: location.ProfileID, Bucket: location.Bucket, Key: key},
			})
			if err != nil {
				return err
			}
			results = append(results, result)
			return nil
		})
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// uploadData stages in-memory content, such as a pasted image, and uploads it
// through the transfer engine so overwrites still require a plan.
func (a *App) uploadData(ctx context.Context, ref core.ObjectRef, data []byte) (service.TransferResult, error) {
	if len(data) == 0 || len(data) > maxUploadData {
		return service.TransferResult{}, core.Invalid("Upload data must be between 1 byte and 64 MiB")
	}
	dir := a.stagedDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return service.TransferResult{}, core.NewError("local_io", "Cannot create the upload staging directory")
	}
	filename := filepath.Join(dir, stagedName(ref.Key))
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return service.TransferResult{}, core.NewError("local_io", "Cannot stage the upload data")
	}
	result, err := a.svc.StartTransfer(ctx, core.TransferRequest{
		Direction: "upload", Object: ref, LocalPath: filename,
	})
	if err != nil {
		_ = os.Remove(filename)
	}
	return result, err
}

func (a *App) stagedDir() string { return filepath.Join(a.svc.DB.Dir, "staged") }

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
func (a *App) releaseStaged(task core.TransferTask) {
	if task.Direction != "upload" || (task.Status != "completed" && task.Status != "cancelled") {
		return
	}
	if filepath.Dir(task.LocalPath) == a.stagedDir() {
		_ = os.Remove(task.LocalPath)
	}
}

// sweepStaged removes staged files that no unfinished upload needs, such as
// those left by overwrite plans that were never executed.
func (a *App) sweepStaged(ctx context.Context) {
	entries, err := os.ReadDir(a.stagedDir())
	if err != nil {
		return
	}
	tasks, err := a.svc.Tasks(ctx)
	if err != nil {
		return
	}
	keep := map[string]bool{}
	for _, task := range tasks {
		if task.Direction == "upload" && task.Status != "completed" && task.Status != "cancelled" {
			keep[task.LocalPath] = true
		}
	}
	for _, entry := range entries {
		filename := filepath.Join(a.stagedDir(), entry.Name())
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || keep[filename] || time.Since(info.ModTime()) < stagedMaxAge {
			continue
		}
		_ = os.Remove(filename)
	}
}
