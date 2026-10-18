package transfer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/amigoer/bucky/internal/core"
)

func (t *execution) download(remote core.ObjectStore) error {
	info, err := remote.Stat(t.job.ctx, t.task.Object)
	if err != nil {
		return err
	}
	if info.ETag == "" {
		return core.NewError("conflict", "The remote object does not have a usable ETag")
	}
	if t.task.RemoteETag == "" && t.task.Size == 0 {
		// Downloads learn the remote size when they start, unlike uploads.
		t.task.Size = info.Size
	} else if info.Size != t.task.Size {
		return core.NewError("conflict", "The remote object changed before the download started")
	}
	if t.task.RemoteETag != "" && t.task.RemoteETag != info.ETag {
		return core.NewError("conflict", "The remote object changed since the download was paused")
	}
	t.task.RemoteETag = info.ETag
	if err := t.save(); err != nil {
		return err
	}

	root, file, err := openDownload(t.task)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	defer func() { _ = file.Close() }()

	offset, err := resumeOffset(file, t.task)
	if err != nil {
		return err
	}
	h := sha256.New()
	if offset > 0 {
		if _, err := io.CopyBuffer(h, io.NewSectionReader(file, 0, offset), make([]byte, 256<<10)); err != nil {
			return localError("Cannot read the partial download")
		}
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return localError("Cannot seek the partial download")
	}
	if err := file.Truncate(offset); err != nil {
		return localError("Cannot truncate the partial download")
	}
	t.base, t.task.Completed = offset, offset
	if err := t.progress(offset, true); err != nil {
		return err
	}
	if offset == t.task.Size {
		if err := finalizeDownload(root, file, t.task); err != nil {
			return err
		}
		t.task.Completed, t.task.Verification = info.Size, "remote-size-etag+local-sha256"
		t.task.SourceSHA256 = hex.EncodeToString(h.Sum(nil))
		return nil
	}

	reader, err := remote.Read(t.job.ctx, t.task.Object, core.ReadOptions{Offset: offset, IfMatch: info.ETag})
	if err != nil {
		return err
	}
	if reader == nil || reader.Body == nil {
		return core.NewError("protocol", "The storage driver returned no object body")
	}
	if reader.Info.Size != info.Size || (reader.Info.ETag != "" && reader.Info.ETag != info.ETag) {
		_ = reader.Body.Close()
		return core.NewError("conflict", "The remote object changed while the download was opening")
	}
	writer := &downloadWriter{
		ctx: t.job.ctx, writer: io.MultiWriter(file, h), limiter: &t.engine.rate, base: offset,
		progress: func(n int64) error { return t.progress(n, false) },
	}
	_, copyErr := io.CopyBuffer(writer, &contextReader{ctx: t.job.ctx, reader: reader.Body}, make([]byte, 256<<10))
	closeErr := reader.Body.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	if offset+writer.count != info.Size {
		return core.NewError("verification_failed", "The download did not receive the expected number of bytes")
	}
	current, err := remote.Stat(t.job.ctx, t.task.Object)
	if err != nil {
		return err
	}
	if current.Size != info.Size || current.ETag != info.ETag {
		return core.NewError("conflict", "The remote object changed during the download")
	}
	if err := finalizeDownload(root, file, t.task); err != nil {
		return err
	}
	t.task.Completed, t.task.Verification = info.Size, "remote-size-etag+local-sha256"
	t.task.SourceSHA256 = hex.EncodeToString(h.Sum(nil))
	return nil
}

func resumeOffset(file *os.File, task core.TransferTask) (int64, error) {
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return 0, localError("Cannot inspect the partial download")
	}
	if info.Size() > task.Size {
		return 0, localError("The partial download is larger than the remote object")
	}
	if info.Size() > 0 && task.RemoteETag == "" {
		return 0, localError("A partial download cannot be resumed without a remote ETag")
	}
	return info.Size(), nil
}

type downloadWriter struct {
	ctx      context.Context
	writer   io.Writer
	limiter  *limiter
	base     int64
	count    int64
	progress func(int64) error
}

func (w *downloadWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.writer.Write(p)
	if n > 0 {
		if waitErr := w.limiter.wait(w.ctx, n); waitErr != nil {
			return n, waitErr
		}
		w.count += int64(n)
		if progressErr := w.progress(w.base + w.count); progressErr != nil {
			return n, progressErr
		}
	}
	return n, err
}
