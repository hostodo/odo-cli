//go:build !windows

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

type poolRetryFileInfoOwner struct {
	os.FileInfo
	uid uint32
}

func (info poolRetryFileInfoOwner) Sys() any {
	stat := *info.FileInfo.Sys().(*syscall.Stat_t)
	stat.Uid = info.uid
	return &stat
}

func TestPoolRetryRejectsPathOwnedByAnotherUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retry-path")
	if err := os.WriteFile(path, []byte("record"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	otherUID := uint32(os.Geteuid()) + 1
	err = checkPoolRetryOwnership(path, poolRetryFileInfoOwner{FileInfo: info, uid: otherUID})
	if err == nil || !strings.Contains(err.Error(), "owned by the current user") {
		t.Fatalf("foreign-owned path accepted: %v", err)
	}
}
