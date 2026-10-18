package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

func (s *Service) StartTransfer(ctx context.Context, req core.TransferRequest) (TransferResult, error) {
	if err := s.validateRef(ctx, req.Object, true); err != nil {
		return TransferResult{}, err
	}
	if req.Direction != "upload" && req.Direction != "download" {
		return TransferResult{}, core.Invalid("The transfer direction must be upload or download")
	}
	if req.Direction == "upload" {
		if err := s.checkWritable(ctx, req.Object.ProfileID); err != nil {
			return TransferResult{}, err
		}
		_, err := s.Stat(ctx, req.Object)
		if err == nil {
			plan, err := s.CreatePlan(ctx, core.PlanRequest{Operation: "upload", Target: req.Object, LocalPath: req.LocalPath})
			return TransferResult{Plan: &plan}, err
		}
		if core.AsError(err).Code != "not_found" {
			return TransferResult{}, err
		}
	}
	options := core.PutOptions{IfNoneMatch: true, ContentType: mime.TypeByExtension(filepath.Ext(req.LocalPath))}
	task, err := s.Transfers.Start(ctx, req, options)
	return TransferResult{Task: &task}, err
}

func (s *Service) ResumeTransfer(ctx context.Context, id string) (core.TransferTask, error) {
	t, err := s.DB.Task(ctx, id)
	if err != nil {
		return t, err
	}
	if err := s.validateRef(ctx, t.Object, true); err != nil {
		return t, err
	}
	if t.Direction == "upload" {
		if err := s.checkWritable(ctx, t.Object.ProfileID); err != nil {
			return t, err
		}
	}
	return s.Transfers.Resume(ctx, id)
}

func (s *Service) CreatePlan(ctx context.Context, req core.PlanRequest) (core.OperationPlan, error) {
	p := core.OperationPlan{ID: ID(), Operation: req.Operation, Status: "pending", Items: []core.PlanItem{}, Results: []core.ItemResult{}, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(15 * time.Minute), Warning: "Object operations are not transactional. Source and destination states are checked again before execution. Where the provider cannot condition a write, another client can change the object between this check and the request."}
	if req.Operation != "delete" && req.Operation != "copy" && req.Operation != "move" && req.Operation != "upload" {
		return p, core.Invalid("Unsupported plan operation")
	}
	if req.Operation == "upload" {
		if err := s.checkWritable(ctx, req.Target.ProfileID); err != nil {
			return p, err
		}
		if err := s.validateRef(ctx, req.Target, true); err != nil {
			return p, err
		}
		info, err := localFile(req.LocalPath)
		if err != nil {
			return p, err
		}
		filename, err := filepath.Abs(req.LocalPath)
		if err != nil {
			return p, core.Invalid("Invalid local file path")
		}
		fingerprint, err := fileHash(ctx, filename)
		if err != nil {
			return p, err
		}
		existing, err := s.existing(ctx, req.Target)
		if err != nil {
			return p, err
		}
		p.Items = append(p.Items, core.PlanItem{Target: &req.Target, Existing: existing, LocalPath: filename, LocalSize: info.Size(), LocalMtime: info.ModTime().UnixNano(), LocalSHA256: fingerprint})
		p.Bytes = info.Size()
	} else {
		if len(req.Sources) == 0 || len(req.Sources) > 1000 {
			return p, core.Invalid("A plan must contain between 1 and 1000 explicit objects")
		}
		seen := map[core.ObjectRef]bool{}
		targets := map[core.ObjectRef]bool{}
		for _, ref := range req.Sources {
			if seen[ref] {
				return p, core.Invalid("A source object is listed more than once")
			}
			seen[ref] = true
			if err := s.checkWritable(ctx, ref.ProfileID); err != nil {
				return p, err
			}
			info, err := s.Stat(ctx, ref)
			if err != nil {
				return p, err
			}
			item := core.PlanItem{Source: info}
			if req.Operation != "delete" {
				target := req.Target
				if target.ProfileID != ref.ProfileID {
					return p, core.NewError("unsupported", "Copy and move require the same connection")
				}
				if len(req.Sources) > 1 {
					if target.Key != "" && target.Key[len(target.Key)-1] != '/' {
						return p, core.Invalid("A multi-object destination must end with '/'")
					}
					target.Key += path.Base(ref.Key)
				}
				if err := s.validateRef(ctx, target, true); err != nil {
					return p, err
				}
				if target == ref || targets[target] {
					return p, core.Invalid("Source and destination objects must be distinct")
				}
				targets[target] = true
				item.Target = &target
				item.Existing, err = s.existing(ctx, target)
				if err != nil {
					return p, err
				}
			}
			p.Items = append(p.Items, item)
			p.Bytes += info.Size
		}
		for target := range targets {
			if seen[target] {
				return p, core.Invalid("A destination cannot also be a source in the same plan")
			}
		}
	}
	return p, s.DB.SavePlan(ctx, p)
}

func (s *Service) existing(ctx context.Context, ref core.ObjectRef) (*core.ObjectInfo, error) {
	info, err := s.Stat(ctx, ref)
	if err != nil {
		if core.AsError(err).Code == "not_found" {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

func (s *Service) ExecutePlan(ctx context.Context, id string, confirmed bool) (core.OperationPlan, error) {
	p, err := s.DB.Plan(ctx, id)
	if err != nil {
		return p, err
	}
	if !confirmed {
		return p, core.NewError("confirmation_required", "Review the plan and explicitly authorize execution")
	}
	if p.Status != "pending" {
		return p, nil
	}
	if time.Now().After(p.ExpiresAt) {
		return p, core.NewError("expired", "The plan expired; generate and review a new plan")
	}
	err = s.Transfers.WithWriteLock(ctx, func() error {
		for _, item := range p.Items {
			profileID := item.Source.ProfileID
			if item.Target != nil {
				profileID = item.Target.ProfileID
			}
			if err := s.checkWritable(ctx, profileID); err != nil {
				return err
			}
			if err := s.checkPlanItem(ctx, p.Operation, item); err != nil {
				return err
			}
		}
		if err := s.DB.ClaimPlan(ctx, p); err != nil {
			return err
		}
		p.Status = "running"
		if p.Operation == "upload" {
			return nil
		}
		for _, item := range p.Items {
			result := core.ItemResult{Source: item.Source.ObjectRef, Target: item.Target, Status: "completed"}
			if err := s.executeItem(ctx, p.Operation, item); err != nil {
				result.Status, result.Error = "failed", core.AsError(err)
			}
			p.Results = append(p.Results, result)
			if err := s.DB.SavePlan(context.WithoutCancel(ctx), p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return p, err
	}
	if p.Operation == "upload" {
		item := p.Items[0]
		opts := targetOptions(item)
		opts.ContentType = mime.TypeByExtension(filepath.Ext(item.LocalPath))
		task, err := s.Transfers.Start(ctx, core.TransferRequest{Direction: "upload", Object: *item.Target, LocalPath: item.LocalPath, ExpectedSHA256: item.LocalSHA256}, opts)
		result := core.ItemResult{Target: item.Target, Status: "queued"}
		if err != nil {
			result.Status, result.Error = "failed", core.AsError(err)
		} else {
			result.Status = "task:" + task.ID
		}
		p.Results = append(p.Results, result)
	}
	p.Status = "completed"
	for _, result := range p.Results {
		if result.Error != nil {
			p.Status = "partial"
		}
	}
	return p, s.DB.SavePlan(context.WithoutCancel(ctx), p)
}

func (s *Service) checkPlanItem(ctx context.Context, operation string, item core.PlanItem) error {
	if operation == "upload" {
		info, err := localFile(item.LocalPath)
		if err != nil || info.Size() != item.LocalSize || info.ModTime().UnixNano() != item.LocalMtime {
			return core.NewError("conflict", "The local source changed after the plan was created")
		}
		hash, err := fileHash(ctx, item.LocalPath)
		if err != nil {
			return err
		}
		if hash != item.LocalSHA256 {
			return core.NewError("conflict", "The local source changed after the plan was created")
		}
	} else {
		info, err := s.Stat(ctx, item.Source.ObjectRef)
		if err != nil {
			return err
		}
		if !sameObject(info, item.Source) {
			return core.NewError("conflict", "A source object changed after the plan was created")
		}
	}
	if item.Target != nil {
		current, err := s.existing(ctx, *item.Target)
		if err != nil {
			return err
		}
		if (current == nil) != (item.Existing == nil) || (current != nil && !sameObject(*current, *item.Existing)) {
			return core.NewError("conflict", "A destination object changed after the plan was created")
		}
	}
	return nil
}

func (s *Service) executeItem(ctx context.Context, operation string, item core.PlanItem) error {
	if err := s.checkPlanItem(ctx, operation, item); err != nil {
		return err
	}
	driver, err := s.resolveWrite(ctx, item.Source.ProfileID)
	if err != nil {
		return err
	}
	if operation != "delete" {
		copier, ok := driver.(core.CopyStore)
		if !ok {
			return core.NewError("unsupported", "Server-side copying is unavailable")
		}
		copied, err := copier.Copy(ctx, item.Source, *item.Target, targetOptions(item))
		if err != nil {
			return err
		}
		if copied.Size != item.Source.Size {
			return core.NewError("verification_failed", "The copied object size does not match; the source was preserved")
		}
		if operation == "copy" {
			return nil
		}
		current, err := driver.Stat(ctx, item.Source.ObjectRef)
		if err != nil || !sameObject(current, item.Source) {
			return core.NewError("conflict", "The source changed during copying; it was preserved")
		}
	}
	results, err := driver.Delete(ctx, []core.ObjectInfo{item.Source})
	if err != nil {
		return err
	}
	if len(results) != 1 {
		return core.NewError("protocol", "The storage service returned an incomplete deletion result")
	}
	if results[0].Error != nil {
		return results[0].Error
	}
	return nil
}

func sameObject(a, b core.ObjectInfo) bool {
	return a.Size == b.Size && a.ETag == b.ETag && a.LastModified.Equal(b.LastModified)
}

func targetOptions(item core.PlanItem) core.PutOptions {
	if item.Existing != nil {
		return core.PutOptions{IfMatch: item.Existing.ETag}
	}
	return core.PutOptions{IfNoneMatch: true}
}

func fileHash(ctx context.Context, filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", core.NewError("local_io", "Cannot read the local source file")
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	buffer := make([]byte, 256*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", core.AsError(err)
		}
		n, err := f.Read(buffer)
		if n > 0 {
			_, _ = h.Write(buffer[:n])
		}
		if err == io.EOF {
			return hex.EncodeToString(h.Sum(nil)), nil
		}
		if err != nil {
			return "", core.NewError("local_io", "Cannot read the local source file")
		}
	}
}
