package cmd

import (
	"reflect"
	"testing"
)

func TestBuildSSHArgs_NoPasswordFallback(t *testing.T) {
	args := buildSSHArgs("user@1.2.3.4", false, nil)
	expected := []string{"user@1.2.3.4"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("buildSSHArgs() = %v, want %v", args, expected)
	}
}

func TestBuildSSHArgs_WithPasswordFallback(t *testing.T) {
	args := buildSSHArgs("root@10.0.0.1", true, nil)
	expected := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"root@10.0.0.1",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("buildSSHArgs() = %v, want %v", args, expected)
	}
}

func TestBuildSSHArgs_WithExtraArgs(t *testing.T) {
	extra := []string{"-L", "8080:localhost:8080", "-v"}
	args := buildSSHArgs("user@1.2.3.4", false, extra)
	expected := []string{"user@1.2.3.4", "-L", "8080:localhost:8080", "-v"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("buildSSHArgs() = %v, want %v", args, expected)
	}
}

func TestBuildSSHArgs_PasswordFallbackWithExtraArgs(t *testing.T) {
	extra := []string{"-D", "1080", "-N"}
	args := buildSSHArgs("root@10.0.0.1", true, extra)
	expected := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"root@10.0.0.1",
		"-D", "1080", "-N",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("buildSSHArgs() = %v, want %v", args, expected)
	}
}

func TestBuildSSHArgs_EmptyExtraArgs(t *testing.T) {
	args := buildSSHArgs("user@host", false, []string{})
	expected := []string{"user@host"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("buildSSHArgs() = %v, want %v", args, expected)
	}
}
