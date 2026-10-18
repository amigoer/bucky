//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package transfer

import (
	"os"
	"syscall"
)

func openNoFollow(root *os.Root, name string, flags int, mode os.FileMode) (*os.File, error) {
	return root.OpenFile(name, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, mode)
}
