package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// BackupSource 抽象出创建一致性快照所需的数据库能力，便于用 *store.Store 实现并解耦测试。
type BackupSource interface {
	Backup(destPath string) (int64, error)
	Ping() error
}

// Info 描述一份备份元数据，作为保留策略与恢复演练的可审计证据。
type Info struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	Schema     int       `json:"schema_version"`
	Integrity  string    `json:"integrity"`
	TableCount int       `json:"table_count"`
}

// DrillResult 是恢复演练的结果，证明最新备份可以被恢复并用于查询。
type DrillResult struct {
	BackupPath   string `json:"backup_path"`
	RestoredPath string `json:"restored_path"`
	Schema       int    `json:"schema_version"`
	TableCount   int    `json:"table_count"`
}

// Manager 负责自动备份、保留策略与恢复验证。
type Manager struct {
	source BackupSource
	dir    string
	retain int
}

// New 创建一个备份管理器。
func New(source BackupSource, dir string, retain int) *Manager {
	return &Manager{source: source, dir: dir, retain: retain}
}

// BackupOnce 执行一次备份：创建快照到备份目录、校验其可读性，然后按保留策略清理旧备份。
func (m *Manager) BackupOnce() (Info, error) {
	if m.source == nil {
		return Info{}, fmt.Errorf("备份数据源未初始化")
	}
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return Info{}, fmt.Errorf("创建备份目录失败: %w", err)
	}
	path := m.nextBackupPath()
	if _, err := m.source.Backup(path); err != nil {
		return Info{}, err
	}
	info, err := m.Verify(path)
	if err != nil {
		return Info{}, fmt.Errorf("备份创建后校验失败: %w", err)
	}
	if _, err := m.Prune(); err != nil {
		return Info{}, fmt.Errorf("清理旧备份失败: %w", err)
	}
	return info, nil
}

// nextBackupPath 以 UTC 时间戳生成唯一备份路径，若同秒冲突则追加序号避免覆盖。
func (m *Manager) nextBackupPath() string {
	stamp := time.Now().UTC().Format("20060102T150405")
	base := filepath.Join(m.dir, "event-go-"+stamp)
	path := base + ".db"
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = fmt.Sprintf("%s-%d.db", base, i)
	}
}

// Prune 保留最新 retain 份事件备份，删除更旧的。返回被删除的文件列表。
func (m *Manager) Prune() ([]string, error) {
	files, err := m.listBackups()
	if err != nil {
		return nil, err
	}
	if m.retain <= 0 || len(files) <= m.retain {
		return nil, nil
	}
	removed := make([]string, 0, len(files)-m.retain)
	for _, f := range files[:len(files)-m.retain] {
		if err := os.Remove(f); err != nil {
			return removed, fmt.Errorf("删除旧备份失败: %w", err)
		}
		removed = append(removed, f)
	}
	return removed, nil
}

// List 返回当前保留的事件备份元数据，按创建时间升序。
func (m *Manager) List() ([]Info, error) {
	files, err := m.listBackups()
	if err != nil {
		return nil, err
	}
	infos := make([]Info, 0, len(files))
	for _, f := range files {
		info, err := m.Verify(f)
		if err != nil {
			return nil, fmt.Errorf("读取备份元数据失败 %s: %w", f, err)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Verify 以只读方式打开备份，运行完整性检查并读取 schema 版本与表数量。
func (m *Manager) Verify(path string) (Info, error) {
	info := Info{Path: path}
	ro, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return info, fmt.Errorf("打开备份失败: %w", err)
	}
	defer ro.Close()

	var integrity string
	if err := ro.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		return info, fmt.Errorf("备份完整性检查失败: %w", err)
	}
	info.Integrity = integrity
	if integrity != "ok" {
		return info, fmt.Errorf("备份完整性异常: %s", integrity)
	}

	var schema int
	if err := ro.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&schema); err != nil {
		return info, fmt.Errorf("备份缺少 schema_migrations 表: %w", err)
	}
	info.Schema = schema

	var tableCount int
	if err := ro.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'`).Scan(&tableCount); err != nil {
		return info, fmt.Errorf("读取备份表数量失败: %w", err)
	}
	info.TableCount = tableCount

	if stat, err := os.Stat(path); err == nil {
		info.Size = stat.Size()
	}
	info.CreatedAt = parseBackupTime(path)
	return info, nil
}

// RestoreDrill 执行一次恢复演练：把最新备份复制到临时位置，以可读写方式打开并校验，证明其可恢复。演练后清理临时文件。
func (m *Manager) RestoreDrill() (DrillResult, error) {
	files, err := m.listBackups()
	if err != nil {
		return DrillResult{}, err
	}
	if len(files) == 0 {
		return DrillResult{}, fmt.Errorf("没有可用于恢复演练的备份")
	}
	latest := files[len(files)-1]
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return DrillResult{}, fmt.Errorf("创建备份目录失败: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102T150405")
	scratch := filepath.Join(m.dir, "drill-"+stamp+".db")
	if err := m.RestoreTo(latest, scratch); err != nil {
		return DrillResult{}, err
	}
	defer os.Remove(scratch)

	info, err := m.Verify(scratch)
	if err != nil {
		return DrillResult{}, fmt.Errorf("恢复演练校验失败: %w", err)
	}
	return DrillResult{
		BackupPath:   latest,
		RestoredPath: scratch,
		Schema:       info.Schema,
		TableCount:   info.TableCount,
	}, nil
}

// RestoreTo 把一份已验证的备份复制到目标路径（用于实际恢复或演练），目标路径存在时拒绝覆盖。
func (m *Manager) RestoreTo(src, dst string) error {
	if src == "" || dst == "" {
		return fmt.Errorf("恢复源与目标路径不能为空")
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("待恢复备份不存在: %w", err)
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("恢复目标已存在，拒绝覆盖: %s", dst)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查恢复目标失败: %w", err)
	}
	if _, err := m.Verify(src); err != nil {
		return fmt.Errorf("恢复前源备份校验失败: %w", err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("写入恢复文件失败: %w", err)
	}
	return nil
}

// Run 启动后台调度：按 interval 定期备份，并按 drillEvery 定期执行恢复演练。
func (m *Manager) Run(ctx context.Context, interval, drillEvery time.Duration, reportError func(error)) {
	report := func(err error) {
		if err != nil && reportError != nil {
			reportError(err)
		}
	}

	backupOnce := func() {
		if _, err := m.BackupOnce(); err != nil {
			report(err)
		}
	}
	drillOnce := func() {
		if _, err := m.RestoreDrill(); err != nil {
			report(err)
		}
	}

	backupOnce()
	if drillEvery > 0 {
		drillOnce()
	}

	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lastDrill := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			backupOnce()
			if drillEvery > 0 && time.Since(lastDrill) >= drillEvery {
				drillOnce()
				lastDrill = time.Now()
			}
		}
	}
}

// listBackups 返回事件备份文件路径列表（排除 drill-* 的临时文件），按名字升序（即创建时间升序）。
func (m *Manager) listBackups() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取备份目录失败: %w", err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "event-go-") || !strings.HasSuffix(name, ".db") {
			continue
		}
		files = append(files, filepath.Join(m.dir, name))
	}
	sort.Strings(files)
	return files, nil
}

// parseBackupTime 从备份文件名解析创建时间；命名不合法时回退到零值。
func parseBackupTime(path string) time.Time {
	base := filepath.Base(path)
	base = strings.TrimPrefix(base, "event-go-")
	base = strings.TrimSuffix(base, ".db")
	if idx := strings.Index(base, "-"); idx >= 0 {
		base = base[:idx]
	}
	parsed, err := time.Parse("20060102T150405", base)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
