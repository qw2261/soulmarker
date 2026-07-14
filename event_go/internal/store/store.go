package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	_ "modernc.org/sqlite"
)

const CurrentSchemaVersion = 3

type Store struct {
	db *sql.DB
}

type migration struct {
	version int
	name    string
	apply   func(*sql.Tx) error
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) Ping() error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return s.db.Ping()
}

// OpenStore 打开数据库并执行版本化迁移，任何初始化失败都会返回给调用方。
func OpenStore(dbPath string) (*Store, error) {
	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	closeOnError := func(err error) (*Store, error) {
		_ = db.Close()
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return closeOnError(fmt.Errorf("连接数据库失败: %w", err))
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return closeOnError(fmt.Errorf("设置 WAL 模式失败: %w", err))
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return closeOnError(fmt.Errorf("设置 busy_timeout 失败: %w", err))
	}
	if err := migrate(db); err != nil {
		return closeOnError(fmt.Errorf("数据库迁移失败: %w", err))
	}

	return &Store{db: db}, nil
}

// NewStore 保留给现有调用方；生产入口使用 OpenStore 显式处理错误。
func NewStore(dbPath string) *Store {
	store, err := OpenStore(dbPath)
	if err != nil {
		panic(err)
	}
	return store
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("创建迁移记录表失败: %w", err)
	}

	applied := make(map[int]bool)
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("读取迁移版本失败: %w", err)
	}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			_ = rows.Close()
			return fmt.Errorf("解析迁移版本失败: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("关闭迁移版本结果失败: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历迁移版本失败: %w", err)
	}

	for _, m := range migrations() {
		if applied[m.version] {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开启迁移 %d 事务失败: %w", m.version, err)
		}
		if err := m.apply(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("执行迁移 %d (%s) 失败: %w", m.version, m.name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, time.Now().UTC().Format(model.TimeFormat),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("记录迁移 %d 失败: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交迁移 %d 失败: %w", m.version, err)
		}
	}
	return nil
}

func migrations() []migration {
	return []migration{
		{version: 1, name: "v5_1_baseline", apply: migrateV51Baseline},
		{version: 2, name: "v5_1_legacy_columns", apply: migrateV51LegacyColumns},
		{version: 3, name: "identity_user_id_expand", apply: migrateIdentityUserIDExpand},
	}
}

func migrateV51Baseline(tx *sql.Tx) error {
	return execStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS organizers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			contact TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			address TEXT NOT NULL DEFAULT '',
			website TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organizer_id INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			event_time TEXT NOT NULL,
			location TEXT NOT NULL,
			capacity INTEGER NOT NULL DEFAULT 0,
			price REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'published',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (organizer_id) REFERENCES organizers(id)
		)`,
		`CREATE TABLE IF NOT EXISTS registrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			contact TEXT NOT NULL,
			ticket_id INTEGER REFERENCES tickets(id),
			ticket_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			author_name TEXT NOT NULL,
			author_contact TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE TABLE IF NOT EXISTS replies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			author_name TEXT NOT NULL,
			author_contact TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (post_id) REFERENCES posts(id)
		)`,
		`CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			price REAL NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			contact TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_registrations_event ON registrations(event_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_registrations_event_contact ON registrations(event_id, contact)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_event ON posts(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_replies_post ON replies(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_event ON tickets(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_status ON events(status)`,
		`CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_event_created ON posts(event_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_registrations_event_created ON registrations(event_id, created_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_contact ON users(contact)`,
	})
}

func migrateV51LegacyColumns(tx *sql.Tx) error {
	columns := []struct {
		table      string
		column     string
		definition string
	}{
		{"registrations", "ticket_id", "ticket_id INTEGER REFERENCES tickets(id)"},
		{"registrations", "ticket_name", "ticket_name TEXT NOT NULL DEFAULT ''"},
		{"events", "organizer_id", "organizer_id INTEGER NOT NULL DEFAULT 0"},
	}
	for _, column := range columns {
		if err := addColumnIfMissing(tx, column.table, column.column, column.definition); err != nil {
			return err
		}
	}
	return execStatements(tx, []string{
		`CREATE INDEX IF NOT EXISTS idx_events_organizer ON events(organizer_id)`,
	})
}

func migrateIdentityUserIDExpand(tx *sql.Tx) error {
	columns := []struct {
		table      string
		column     string
		definition string
	}{
		{"registrations", "user_id", "user_id INTEGER REFERENCES users(id)"},
		{"posts", "user_id", "user_id INTEGER REFERENCES users(id)"},
		{"replies", "user_id", "user_id INTEGER REFERENCES users(id)"},
	}
	for _, column := range columns {
		if err := addColumnIfMissing(tx, column.table, column.column, column.definition); err != nil {
			return err
		}
	}
	return execStatements(tx, []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_registrations_event_user ON registrations(event_id, user_id) WHERE user_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_posts_user ON posts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_replies_user ON replies(user_id)`,
	})
}

func execStatements(tx *sql.Tx, statements []string) error {
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func addColumnIfMissing(tx *sql.Tx, table, column, definition string) error {
	exists, err := tableHasColumn(tx, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition)); err != nil {
		return fmt.Errorf("为 %s 增加 %s 列失败: %w", table, column, err)
	}
	return nil
}

func tableHasColumn(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("读取 %s 表结构失败: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("解析 %s 表结构失败: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("遍历 %s 表结构失败: %w", table, err)
	}
	return false, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
}
