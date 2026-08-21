package buildinfo

import "testing"

func TestDefaultVersionNonEmpty(t *testing.T) {
	if got := DefaultVersion(); got == "" {
		t.Fatal("DefaultVersion should not be empty")
	}
}

func TestResultPopulatesCoreFields(t *testing.T) {
	if Version == "" {
		t.Fatal("Version variable must be non-empty")
	}
	if Commit == "" {
		t.Fatal("Commit variable must be non-empty")
	}

	info := Result()
	if info.Version == "" {
		t.Fatal("info.Version should not be empty")
	}
	if info.Commit == "" {
		t.Fatal("info.Commit should not be empty")
	}
	if info.GoVersion == "" {
		t.Fatal("info.GoVersion should not be empty")
	}
	if info.BuildTime == "" {
		t.Fatal("info.BuildTime should not be empty")
	}
}

func TestResultSourceVersionMatchesVar(t *testing.T) {
	// 注入变量后，Result.Version 应与包级 Version 一致（此测试无 ldflags 时两者均为 "dev"）。
	info := Result()
	if info.Version != Version {
		t.Fatalf("info.Version = %q, want %q", info.Version, Version)
	}
	if info.Commit != Commit {
		t.Fatalf("info.Commit = %q, want %q", info.Commit, Commit)
	}
}

func TestResultReadsBuildInfo(t *testing.T) {
	info := Result()
	// go test 二进制包含模块与依赖的 build info，至少应解析出 module 或依赖之一。
	if info.Module == "" && len(info.Dependencies) == 0 {
		t.Fatal("expected module or dependencies from build info")
	}
}
