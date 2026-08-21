package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/store"
)

func newTestSource(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.OpenStore(filepath.Join(dir, "source.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, dir
}

func TestBackupOnceCreatesVerifiedSnapshot(t *testing.T) {
	s, dir := newTestSource(t)
	m := New(s, filepath.Join(dir, "backups"), 7)

	info, err := m.BackupOnce()
	if err != nil {
		t.Fatalf("backup once: %v", err)
	}
	if info.Schema != store.CurrentSchemaVersion {
		t.Errorf("schema = %d, want %d", info.Schema, store.CurrentSchemaVersion)
	}
	if info.Integrity != "ok" {
		t.Errorf("integrity = %q, want ok", info.Integrity)
	}
	if info.Size <= 0 {
		t.Errorf("backup size should be > 0, got %d", info.Size)
	}
	if info.TableCount == 0 {
		t.Errorf("expected tables in backup, got 0")
	}
	if _, err := os.Stat(info.Path); err != nil {
		t.Errorf("backup file missing: %v", err)
	}
	if info.CreatedAt.IsZero() {
		t.Errorf("expected parseable created_at from filename")
	}
}

func TestPruneEnforcesRetention(t *testing.T) {
	s, dir := newTestSource(t)
	m := New(s, filepath.Join(dir, "backups"), 3)

	for i := 0; i < 5; i++ {
		if _, err := m.BackupOnce(); err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
	}
	infos, err := m.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("retention kept %d backups, want 3", len(infos))
	}
}

func TestRestoreDrillRestoresLatest(t *testing.T) {
	s, dir := newTestSource(t)
	m := New(s, filepath.Join(dir, "backups"), 7)

	if _, err := m.BackupOnce(); err != nil {
		t.Fatalf("backup once: %v", err)
	}
	result, err := m.RestoreDrill()
	if err != nil {
		t.Fatalf("restore drill: %v", err)
	}
	if result.BackupPath == "" || result.RestoredPath == "" {
		t.Fatalf("drill result missing paths: %+v", result)
	}
	if result.Schema != store.CurrentSchemaVersion {
		t.Errorf("drill schema = %d, want %d", result.Schema, store.CurrentSchemaVersion)
	}
	if result.TableCount == 0 {
		t.Errorf("drill restored db has no tables")
	}
	// 演练结束后应清理临时恢复文件
	entries, _ := os.ReadDir(filepath.Dir(result.RestoredPath))
	if _, err := os.Stat(result.RestoredPath); !os.IsNotExist(err) {
		t.Errorf("drill scratch file should be removed")
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "drill-") {
			t.Errorf("leftover drill file: %s", e.Name())
		}
	}
}

func TestVerifyRejectsCorruptBackup(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(corrupt, []byte("this is not a sqlite database"), 0o600); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	m := New(nil, dir, 7)
	if _, err := m.Verify(corrupt); err == nil {
		t.Fatalf("expected error verifying corrupt backup, got nil")
	}
}

func TestRestoreToRefusesOverwrite(t *testing.T) {
	s, dir := newTestSource(t)
	backupDir := filepath.Join(dir, "backups")
	m := New(s, backupDir, 7)
	if _, err := m.BackupOnce(); err != nil {
		t.Fatalf("backup once: %v", err)
	}
	latest := filepath.Join(backupDir, "exists.db")
	if err := os.WriteFile(latest, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("create target: %v", err)
	}
	list, err := m.List()
	if err != nil || len(list) == 0 {
		t.Fatalf("list: %v (count=%d)", err, len(list))
	}
	if err := m.RestoreTo(list[len(list)-1].Path, latest); err == nil {
		t.Fatalf("expected error restoring over existing file")
	}
}
