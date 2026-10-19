package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type DesktopService struct {
	backend *service.Service
	cancel  sync.Map
}

func New(backend *service.Service) *DesktopService { return &DesktopService{backend: backend} }

func (d *DesktopService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	d.sweepStaged(ctx)
	events := application.Get().Event
	d.backend.Transfers.OnUpdate = func(task core.TransferTask) {
		events.Emit("transfer:updated", task)
		d.releaseStaged(task)
	}
	return nil
}

func (d *DesktopService) ServiceShutdown() error { return d.backend.Close() }

func (d *DesktopService) Profiles(ctx context.Context) ([]core.Profile, error) {
	return d.backend.Profiles(ctx)
}

func (d *DesktopService) SaveProfile(ctx context.Context, input service.ProfileInput) (core.Profile, error) {
	return d.backend.SaveProfile(ctx, input)
}

func (d *DesktopService) RemoveProfile(ctx context.Context, id string, confirmed bool) error {
	return d.backend.RemoveProfile(ctx, id, confirmed)
}

func (d *DesktopService) TestConnection(ctx context.Context, id string) ([]core.Bucket, error) {
	return d.backend.TestConnection(ctx, id)
}

func (d *DesktopService) TestDraft(ctx context.Context, input service.ProfileInput) ([]core.Bucket, error) {
	return d.backend.TestDraft(ctx, input)
}

func (d *DesktopService) Buckets(ctx context.Context, id string) ([]core.Bucket, error) {
	return d.backend.Buckets(ctx, id)
}

func (d *DesktopService) List(ctx context.Context, req core.ListRequest) (core.ListPage, error) {
	return d.backend.List(ctx, req)
}

func (d *DesktopService) Stat(ctx context.Context, ref core.ObjectRef) (core.ObjectInfo, error) {
	return d.backend.Stat(ctx, ref)
}

func (d *DesktopService) ReadText(ctx context.Context, ref core.ObjectRef) (core.TextPreview, error) {
	return d.backend.ReadText(ctx, ref, 1024*1024)
}

func (d *DesktopService) Share(ctx context.Context, ref core.ObjectRef, seconds int, public bool) (core.ShareLink, error) {
	return d.backend.Share(ctx, ref, int64(seconds), public)
}

func (d *DesktopService) Find(ctx context.Context, req core.FindRequest) (core.FindResult, error) {
	return d.backend.Find(ctx, req)
}

type UsageProgress struct {
	RequestID string     `json:"requestId"`
	Usage     core.Usage `json:"usage"`
}

// Usage returns the partial totals without an error when the request is
// cancelled; Complete stays false.
func (d *DesktopService) Usage(ctx context.Context, id, bucket, prefix, requestID string) (core.Usage, error) {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel.Store(requestID, cancel)
	defer func() { cancel(); d.cancel.Delete(requestID) }()
	events := application.Get().Event
	usage, err := d.backend.ScanUsage(ctx, id, bucket, prefix, func(u core.Usage) {
		events.Emit("usage:progress", UsageProgress{RequestID: requestID, Usage: u})
	})
	if err != nil && ctx.Err() != nil && core.AsError(err).Code == "cancelled" {
		return usage, nil
	}
	return usage, err
}

func (d *DesktopService) CancelRequest(id string) {
	if cancel, ok := d.cancel.Load(id); ok {
		cancel.(context.CancelFunc)()
	}
}

func (d *DesktopService) StartTransfer(ctx context.Context, req core.TransferRequest) (service.TransferResult, error) {
	return d.backend.StartTransfer(ctx, req)
}

func (d *DesktopService) UploadPaths(ctx context.Context, id, bucket, prefix string, paths []string) ([]service.TransferResult, error) {
	results := []service.TransferResult{}
	normalized := strings.Trim(prefix, "/")
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
			result, err := d.backend.StartTransfer(ctx, core.TransferRequest{Direction: "upload", LocalPath: name, Object: core.ObjectRef{ProfileID: id, Bucket: bucket, Key: key}})
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

func (d *DesktopService) Tasks(ctx context.Context) ([]core.TransferTask, error) {
	return d.backend.Transfers.Tasks(ctx)
}

func (d *DesktopService) ResumeTransfer(ctx context.Context, id string) (core.TransferTask, error) {
	return d.backend.ResumeTransfer(ctx, id)
}

func (d *DesktopService) PauseTransfer(ctx context.Context, id string) error {
	return d.backend.Transfers.Pause(ctx, id)
}

func (d *DesktopService) CancelTransfer(ctx context.Context, id string) error {
	return d.backend.Transfers.Cancel(ctx, id)
}

func (d *DesktopService) SetRate(bytesPerSecond int) {
	d.backend.Transfers.SetRate(int64(bytesPerSecond))
}

func (d *DesktopService) SetConcurrency(n int) int {
	return d.backend.Transfers.SetConcurrency(n)
}

func (d *DesktopService) CreatePlan(ctx context.Context, req core.PlanRequest) (core.OperationPlan, error) {
	return d.backend.CreatePlan(ctx, req)
}

func (d *DesktopService) ExecutePlan(ctx context.Context, id string, confirmed bool) (core.OperationPlan, error) {
	return d.backend.ExecutePlan(ctx, id, confirmed)
}

func (d *DesktopService) GetPlan(ctx context.Context, id string) (core.OperationPlan, error) {
	return d.backend.DB.Plan(ctx, id)
}

func (d *DesktopService) PickFiles(directory bool) ([]string, error) {
	dialog := application.Get().Dialog.OpenFile()
	if directory {
		filename, err := dialog.CanChooseFiles(false).CanChooseDirectories(true).PromptForSingleSelection()
		if err != nil {
			return nil, core.NewError("dialog", "Cannot open the directory picker")
		}
		if filename == "" {
			return []string{}, nil
		}
		return []string{filename}, nil
	}
	filenames, err := dialog.CanChooseFiles(true).CanChooseDirectories(false).PromptForMultipleSelection()
	if err != nil {
		return nil, core.NewError("dialog", "Cannot open the file picker")
	}
	if filenames == nil {
		filenames = []string{}
	}
	return filenames, nil
}

func (d *DesktopService) PickDirectory() (string, error) {
	filename, err := application.Get().Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).CanCreateDirectories(true).PromptForSingleSelection()
	if err != nil {
		return "", core.NewError("dialog", "Cannot open the directory picker")
	}
	return filename, nil
}

func (d *DesktopService) PickDownload(name string) (string, error) {
	filename, err := application.Get().Dialog.SaveFile().SetFilename(filepath.Base(strings.ReplaceAll(name, "\\", "/"))).PromptForSingleSelection()
	if err != nil {
		return "", core.NewError("dialog", "Cannot open the save dialog")
	}
	return filename, nil
}

func (d *DesktopService) CopyText(text string) error {
	if !application.Get().Clipboard.SetText(text) {
		return core.NewError("clipboard", "Cannot copy text to the clipboard")
	}
	return nil
}
