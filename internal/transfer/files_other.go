//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly

package transfer

import (
	"errors"
	"os"
)

func openNoFollow(root *os.Root, name string, flags int, mode os.FileMode) (*os.File, error) {
	before, err := root.Lstat(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if before != nil && !before.Mode().IsRegular() {
		return nil, os.ErrPermission
	}
	file, err := root.OpenFile(name, flags, mode)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	current, pathErr := root.Lstat(name)
	if err != nil || pathErr != nil || !after.Mode().IsRegular() || !current.Mode().IsRegular() || !os.SameFile(after, current) || before != nil && !os.SameFile(before, after) {
		_ = file.Close()
		return nil, os.ErrPermission
	}
	return file, nil
}
