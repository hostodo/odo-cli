//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func checkPoolRetryOwnership(path string, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("cache path must be owned by the current user: %s", path)
	}
	return nil
}

func lockPoolRetryFile(file *os.File) error {
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX)
		if err != unix.EINTR {
			return err
		}
	}
}

func checkPoolRetryOwnerPermissions(path string, info os.FileInfo) error {
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("insecure permissions on %s (require owner-only access)", path)
	}
	return nil
}

func syncPoolRetryDirPlatform(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func publishPoolRetryFile(source, destination string) error {
	return os.Link(source, destination)
}
