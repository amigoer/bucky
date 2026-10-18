package transfer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"slices"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

func (t *execution) upload(remote core.ObjectStore) error {
	file, err := openSource(t.task.LocalPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if _, err := verifySource(t.job.ctx, file, t.task); err != nil {
		return err
	}
	if t.task.PutState != "" {
		verified, err := t.verifyRemote(t.job.ctx, remote)
		if err != nil {
			return err
		}
		if verified {
			t.task.PutState = "committed"
			return nil
		}
		t.task.PutState = ""
		if err := t.save(); err != nil {
			return err
		}
	}
	if t.task.Size > multipartThreshold {
		multipart, ok := remote.(core.MultipartStore)
		if !ok || !remote.Capabilities().Multipart {
			return core.NewError("unsupported", "This storage driver does not support multipart uploads")
		}
		return t.multipart(file, remote, multipart)
	}
	return t.singlePut(file, remote)
}

func (t *execution) singlePut(file *os.File, remote core.ObjectStore) error {
	t.task.Completed, t.task.PutState = 0, "writing"
	if err := t.save(); err != nil {
		return err
	}
	h := sha256.New()
	reader := &uploadReader{
		ctx: t.job.ctx, reader: io.TeeReader(io.NewSectionReader(file, 0, t.task.Size), h),
		limiter:  &t.engine.rate,
		progress: func(n int64) error { return t.progress(n, false) },
	}
	info, err := remote.Put(t.job.ctx, t.task.Object, reader, t.task.Size, t.task.PutOptions)
	if err != nil {
		return t.uncertain(remote, err)
	}
	if reader.count != t.task.Size || hex.EncodeToString(h.Sum(nil)) != t.task.SourceSHA256 {
		return core.NewError("source_changed", "The upload source changed during transfer")
	}
	if _, err := verifySource(t.job.ctx, file, t.task); err != nil {
		return err
	}
	if info.Size != t.task.Size || info.ETag == "" {
		return t.uncertain(remote, core.NewError("verification_failed", "The uploaded object metadata does not match the source"))
	}
	t.task.RemoteETag, t.task.Verification, t.task.PutState = info.ETag, "remote-size-etag", "committed"
	return nil
}

func (t *execution) multipart(file *os.File, remote core.ObjectStore, multipart core.MultipartStore) error {
	if t.task.PartSize < minPartSize || (t.task.Size+t.task.PartSize-1)/t.task.PartSize > maxParts {
		return core.NewError("invalid_state", "The saved multipart size is invalid")
	}
	if t.task.UploadID == "" {
		id, err := multipart.CreateMultipart(t.job.ctx, t.task.Object, t.task.PutOptions)
		if err != nil {
			return err
		}
		if id == "" {
			return core.NewError("invalid_state", "The storage driver returned an empty upload ID")
		}
		t.task.UploadID = id
		if err := t.save(); err != nil {
			return err
		}
	}
	parts, err := t.reconcile(file, multipart)
	if err != nil {
		return err
	}
	confirmed := int64(0)
	for _, part := range parts {
		confirmed += part.Size
	}
	t.base, t.task.Completed = confirmed, confirmed
	if err := t.progress(confirmed, true); err != nil {
		return err
	}
	count := int32((t.task.Size + t.task.PartSize - 1) / t.task.PartSize)
	for number := int32(1); number <= count; number++ {
		if err := t.job.ctx.Err(); err != nil {
			return err
		}
		if _, ok := parts[number]; ok {
			continue
		}
		offset := int64(number-1) * t.task.PartSize
		size := min(t.task.PartSize, t.task.Size-offset)
		digest, err := sectionHash(t.job.ctx, file, offset, size)
		if err != nil {
			return err
		}
		h := sha256.New()
		reader := &uploadReader{
			ctx: t.job.ctx, reader: io.TeeReader(io.NewSectionReader(file, offset, size), h),
			limiter:  &t.engine.rate,
			progress: func(n int64) error { return t.progress(confirmed+n, false) },
		}
		part, err := multipart.UploadPart(t.job.ctx, t.task.Object, t.task.UploadID, number, reader, size)
		if err != nil {
			return err
		}
		if reader.count != size || hex.EncodeToString(h.Sum(nil)) != digest {
			return core.NewError("source_changed", "The upload source changed during a multipart request")
		}
		if part.Number != number || part.ETag == "" || part.Size != size {
			return core.NewError("verification_failed", "The uploaded part metadata is invalid")
		}
		part.SHA256 = digest
		parts[number], confirmed = part, confirmed+size
		t.task.Completed = confirmed
		if err := t.savePart(part); err != nil {
			return err
		}
	}
	if _, err := verifySource(t.job.ctx, file, t.task); err != nil {
		return err
	}
	completed := make([]core.Part, 0, len(parts))
	for number := int32(1); number <= count; number++ {
		completed = append(completed, parts[number])
	}
	t.task.PutState = "writing"
	if err := t.save(); err != nil {
		return err
	}
	info, err := multipart.CompleteMultipart(t.job.ctx, t.task.Object, t.task.UploadID, completed, t.task.PutOptions)
	if err != nil {
		return t.uncertain(remote, err)
	}
	if info.Size != t.task.Size || info.ETag == "" {
		return t.uncertain(remote, core.NewError("verification_failed", "The completed upload metadata does not match the source"))
	}
	t.task.RemoteETag, t.task.Verification, t.task.PutState = info.ETag, "remote-size-etag", "committed"
	return nil
}

func (t *execution) reconcile(file *os.File, multipart core.MultipartStore) (map[int32]core.Part, error) {
	local, err := t.engine.db.Parts(t.job.ctx, t.task.ID)
	if err != nil {
		return nil, err
	}
	remote, err := multipart.ListParts(t.job.ctx, t.task.Object, t.task.UploadID)
	if err != nil {
		return nil, err
	}
	saved := make(map[int32]core.Part, len(local))
	for _, part := range local {
		saved[part.Number] = part
	}
	result := make(map[int32]core.Part)
	count := int32((t.task.Size + t.task.PartSize - 1) / t.task.PartSize)
	for _, part := range remote {
		if part.Number < 1 || part.Number > count {
			return nil, core.NewError("conflict", "The multipart upload contains an unexpected part")
		}
		if _, duplicate := result[part.Number]; duplicate {
			return nil, core.NewError("conflict", "The multipart upload contains duplicate parts")
		}
		previous, ok := saved[part.Number]
		offset := int64(part.Number-1) * t.task.PartSize
		size := min(t.task.PartSize, t.task.Size-offset)
		if !ok || part.Size != size || part.ETag == "" || part.ETag != previous.ETag || previous.Size != size || previous.SHA256 == "" {
			continue
		}
		digest, err := sectionHash(t.job.ctx, file, offset, size)
		if err != nil {
			return nil, err
		}
		if digest != previous.SHA256 {
			return nil, core.NewError("source_changed", "The source no longer matches its saved upload parts")
		}
		part.SHA256 = digest
		result[part.Number] = part
	}
	return result, nil
}

func sectionHash(ctx context.Context, file *os.File, offset, size int64) (string, error) {
	h := sha256.New()
	n, err := io.CopyBuffer(h, &contextReader{ctx: ctx, reader: io.NewSectionReader(file, offset, size)}, make([]byte, 256<<10))
	if err != nil {
		return "", err
	}
	if n != size {
		return "", core.NewError("source_changed", "The upload source was truncated")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (t *execution) uncertain(remote core.ObjectStore, original error) error {
	t.task.PutState = "uncertain"
	if err := t.save(); err != nil {
		return errors.Join(original, err)
	}
	if t.job.ctx.Err() != nil {
		return original
	}
	ctx, cancel := context.WithTimeout(t.job.ctx, 30*time.Second)
	defer cancel()
	verified, err := t.verifyRemote(ctx, remote)
	if err == nil && verified {
		t.task.PutState = "committed"
		return nil
	}
	if err != nil {
		return err
	}
	t.task.PutState = ""
	if err := t.save(); err != nil {
		return errors.Join(original, err)
	}
	return original
}

func (t *execution) verifyRemote(ctx context.Context, remote core.ObjectStore) (bool, error) {
	info, err := remote.Stat(ctx, t.task.Object)
	if err != nil {
		if code := core.AsError(err).Code; slices.Contains([]string{"not_found", "object_not_found", "NoSuchKey"}, code) {
			return false, nil
		}
		return false, err
	}
	if info.Size != t.task.Size || info.ETag == "" || info.Metadata["bucky-transfer-id"] != t.task.ID {
		return false, core.NewError("write_uncertain", "The previous write could not be verified; inspect the remote object before creating another transfer")
	}
	reader, err := remote.Read(ctx, t.task.Object, core.ReadOptions{IfMatch: info.ETag})
	if err != nil {
		return false, err
	}
	if reader == nil || reader.Body == nil {
		return false, core.NewError("verification_failed", "The storage driver returned no object body")
	}
	h := sha256.New()
	n, copyErr := io.CopyBuffer(h, io.LimitReader(&contextReader{ctx: ctx, reader: reader.Body}, t.task.Size+1), make([]byte, 256<<10))
	closeErr := reader.Body.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return false, err
	}
	if reader.Info.ETag != info.ETag || n != t.task.Size || hex.EncodeToString(h.Sum(nil)) != t.task.SourceSHA256 {
		return false, core.NewError("write_uncertain", "The remote object does not match the upload fingerprint")
	}
	current, err := remote.Stat(ctx, t.task.Object)
	if err != nil {
		return false, err
	}
	if current.Size != info.Size || current.ETag != info.ETag {
		return false, core.NewError("conflict", "The uploaded object changed during verification")
	}
	t.task.RemoteETag, t.task.Verification, t.task.Completed = info.ETag, "remote-sha256", t.task.Size
	return true, nil
}

func (t *execution) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if t.task.Direction == "upload" {
		if t.task.UploadID == "" {
			return nil
		}
		if err := t.engine.checkWritable(ctx, t.task.Object.ProfileID); err != nil {
			return err
		}
		remote, err := t.engine.resolve(ctx, t.task.Object.ProfileID)
		if err != nil {
			return err
		}
		multipart, ok := remote.(core.MultipartStore)
		if !ok {
			return core.NewError("unsupported", "This storage driver cannot abort the multipart upload")
		}
		if err := multipart.AbortMultipart(ctx, t.task.Object, t.task.UploadID); err != nil {
			return err
		}
		t.task.UploadID = ""
		return nil
	}
	root, _, err := openParent(t.task.LocalPath)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if err := root.Remove(tempName(t.task)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return localError("Cannot remove the cancelled download temporary file")
	}
	return nil
}
