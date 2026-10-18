package transfer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/amigoer/bucky/internal/core"
)

func localError(message string) error {
	return core.NewError("local_file", message)
}

func openParent(path string) (*os.Root, string, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return nil, "", localError("The local path must be absolute")
	}
	parent, name := filepath.Dir(path), filepath.Base(path)
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return nil, "", localError("The local directory is missing")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return nil, "", localError("The local directory is missing")
	}
	root, err := os.OpenRoot(resolved)
	if err != nil {
		return nil, "", localError("Cannot open the local directory")
	}
	return root, name, nil
}

func openSource(path string) (*os.File, error) {
	root, name, err := openParent(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	before, err := root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() {
		return nil, localError("The upload source must be a regular file without symbolic links")
	}
	file, err := openNoFollow(root, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, localError("Cannot open the upload source safely")
	}
	after, statErr := file.Stat()
	if statErr != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, core.NewError("source_changed", "The upload source changed while it was being opened")
	}
	return file, nil
}

func fingerprint(ctx context.Context, file *os.File) (os.FileInfo, string, error) {
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return nil, "", localError("Cannot inspect the upload source")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, "", localError("Cannot seek the upload source")
	}
	h := sha256.New()
	if _, err := io.CopyBuffer(h, &contextReader{ctx: ctx, reader: file}, make([]byte, 256<<10)); err != nil {
		return nil, "", err
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || before.ModTime() != after.ModTime() || !os.SameFile(before, after) {
		return nil, "", core.NewError("source_changed", "The upload source changed while its fingerprint was being calculated")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, "", localError("Cannot seek the upload source")
	}
	return after, hex.EncodeToString(h.Sum(nil)), nil
}

func verifySource(ctx context.Context, file *os.File, task core.TransferTask) (os.FileInfo, error) {
	info, digest, err := fingerprint(ctx, file)
	if err != nil {
		return nil, err
	}
	if info.Size() != task.Size || info.ModTime().UnixNano() != task.SourceModTime || digest != task.SourceSHA256 {
		return nil, core.NewError("source_changed", "The upload source changed; create a new transfer")
	}
	root, name, err := openParent(task.LocalPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	current, err := root.Lstat(name)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(current, info) {
		return nil, core.NewError("source_changed", "The upload source was replaced; create a new transfer")
	}
	return info, nil
}

func checkDestination(path string) error {
	root, name, err := openParent(path)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	_, err = root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err == nil {
		return core.NewError("conflict", "The download destination already exists")
	}
	return localError("Cannot inspect the download destination")
}

func tempName(task core.TransferTask) string {
	return "." + filepath.Base(task.LocalPath) + "." + task.ID + ".bucky-part"
}

func openDownload(task core.TransferTask) (*os.Root, *os.File, error) {
	root, name, err := openParent(task.LocalPath)
	if err != nil {
		return nil, nil, err
	}
	if _, err := root.Lstat(name); err == nil || !errors.Is(err, os.ErrNotExist) {
		_ = root.Close()
		return nil, nil, core.NewError("conflict", "The download destination already exists")
	}
	file, err := openNoFollow(root, tempName(task), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if errors.Is(err, os.ErrExist) {
		info, statErr := root.Lstat(tempName(task))
		if statErr != nil || !info.Mode().IsRegular() || task.RemoteETag == "" {
			_ = root.Close()
			return nil, nil, localError("The download temporary file cannot be resumed safely")
		}
		file, err = openNoFollow(root, tempName(task), os.O_RDWR, 0)
		if err == nil {
			current, statErr := file.Stat()
			if statErr != nil || !current.Mode().IsRegular() || !os.SameFile(info, current) {
				_ = file.Close()
				err = localError("The download temporary file was replaced")
			}
		}
	}
	if err != nil {
		_ = root.Close()
		return nil, nil, localError("Cannot open the download temporary file safely")
	}
	return root, file, nil
}

func finalizeDownload(root *os.Root, file *os.File, task core.TransferTask) error {
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != task.Size {
		return localError("Cannot inspect the completed download")
	}
	current, err := root.Lstat(tempName(task))
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(current, info) {
		return localError("The download temporary file was replaced")
	}
	if err := file.Sync(); err != nil {
		return localError("Cannot flush the completed download")
	}
	// Linking is atomic and fails if any destination already exists.
	if err := root.Link(tempName(task), filepath.Base(task.LocalPath)); err != nil {
		if errors.Is(err, os.ErrExist) {
			return core.NewError("conflict", "The download destination already exists")
		}
		return localError("Cannot finalize the download without overwriting the destination")
	}
	final, err := root.Lstat(filepath.Base(task.LocalPath))
	if err != nil || !final.Mode().IsRegular() || !os.SameFile(final, info) {
		return localError("The finalized download does not match the temporary file")
	}
	if err := root.Remove(tempName(task)); err != nil {
		return localError("Cannot remove the completed download temporary file")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
