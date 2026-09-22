//go:build windows

package cmd

import (
	"os"

	"golang.org/x/sys/windows"
)

func checkPoolRetryOwnership(_ string, _ os.FileInfo) error {
	// Windows profile ACLs, rather than Unix ownership/mode bits, govern access.
	return nil
}

func lockPoolRetryFile(file *os.File) error {
	// os.OpenFile creates a synchronous handle. Without FAIL_IMMEDIATELY,
	// LockFileEx waits for the exclusive byte-range lock, even on an empty file.
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &windows.Overlapped{})
}

func checkPoolRetryOwnerPermissions(_ string, _ os.FileInfo) error {
	// Windows mode bits do not represent access control. Files inherit the
	// user's profile ACL; the shared checks still reject symlinks and devices.
	return nil
}

func syncPoolRetryDirPlatform(_ string) error {
	// Windows does not support Go's directory Sync. Publication below requests
	// write-through after the temporary file's contents have been flushed.
	return nil
}

func publishPoolRetryFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	// Same-directory rename, without REPLACE_EXISTING or COPY_ALLOWED: atomic
	// publication fails if another process already claimed this key.
	return windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH)
}
