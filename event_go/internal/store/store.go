package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/emailaddr"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	_ "modernc.org/sqlite"
)

const CurrentSchemaVersion = 10

type Store struct {
	db             *sql.DB
	registrationMu sync.Mutex
	checkinMu      sync.Mutex
	moderationMu   sync.Mutex
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
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

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
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return closeOnError(fmt.Errorf("启用外键失败: %w", err))
	}
	if err := validateForeignKeys(db); err != nil {
		return closeOnError(err)
	}

	return &Store{db: db}, nil
}

// NewStore 打开数据库并显式返回初始化错误。
func NewStore(dbPath string) (*Store, error) {
	return OpenStore(dbPath)
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
		{version: 4, name: "identity_backfill_and_legacy_status", apply: migrateIdentityBackfill},
		{version: 5, name: "foreign_key_readiness", apply: migrateForeignKeyReadiness},
		{version: 6, name: "admission_and_checkin", apply: migrateAdmissionAndCheckin},
		{version: 7, name: "authentication_session_and_password_reset", apply: migrateAuthenticationSessionAndPasswordReset},
		{version: 8, name: "event_cover_url", apply: migrateEventCoverURL},
		{version: 9, name: "content_moderation", apply: migrateContentModeration},
		{version: 10, name: "recovery_email", apply: migrateRecoveryEmail},
	}
}

func migrateRecoveryEmail(tx *sql.Tx) error {
	columns := []struct {
		column, definition string
	}{
		{"recovery_email", "recovery_email TEXT NOT NULL DEFAULT ''"},
		{"recovery_email_verified_at", "recovery_email_verified_at TEXT"},
	}
	for _, column := range columns {
		if err := addColumnIfMissing(tx, "users", column.column, column.definition); err != nil {
			return err
		}
	}
	if err := execStatements(tx, []string{
		`CREATE TABLE recovery_email_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			used_at TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
	}); err != nil {
		return err
	}
	type recoveryEmailBackfill struct {
		userID     int64
		email      string
		verifiedAt string
	}
	rows, err := tx.Query(`SELECT id, contact, created_at FROM users WHERE recovery_email = ''`)
	if err != nil {
		return fmt.Errorf("查询恢复邮箱回填用户失败: %w", err)
	}
	candidates := make(map[string][]recoveryEmailBackfill)
	for rows.Next() {
		var userID int64
		var contact, createdAt string
		if err := rows.Scan(&userID, &contact, &createdAt); err != nil {
			rows.Close()
			return fmt.Errorf("读取恢复邮箱回填用户失败: %w", err)
		}
		if email, ok := emailaddr.Normalize(contact); ok {
			candidates[email] = append(candidates[email], recoveryEmailBackfill{
				userID: userID, email: email, verifiedAt: createdAt,
			})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("遍历恢复邮箱回填用户失败: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("关闭恢复邮箱回填查询失败: %w", err)
	}
	for _, matches := range candidates {
		if len(matches) != 1 {
			continue
		}
		backfill := matches[0]
		if _, err := tx.Exec(
			`UPDATE users SET recovery_email = ?, recovery_email_verified_at = ? WHERE id = ?`,
			backfill.email, backfill.verifiedAt, backfill.userID,
		); err != nil {
			return fmt.Errorf("回填恢复邮箱失败: %w", err)
		}
	}
	return execStatements(tx, []string{
		`CREATE UNIQUE INDEX idx_users_recovery_email ON users(recovery_email) WHERE recovery_email <> ''`,
		`CREATE INDEX idx_recovery_email_tokens_user ON recovery_email_tokens(user_id, created_at DESC)`,
	})
}

func migrateContentModeration(tx *sql.Tx) error {
	columns := []struct {
		table, column, definition string
	}{
		{"posts", "moderation_status", "moderation_status TEXT NOT NULL DEFAULT 'visible' CHECK (moderation_status IN ('visible', 'removed'))"},
		{"posts", "moderated_at", "moderated_at TEXT"},
		{"posts", "moderated_by", "moderated_by TEXT NOT NULL DEFAULT ''"},
		{"posts", "moderation_reason", "moderation_reason TEXT NOT NULL DEFAULT ''"},
		{"replies", "moderation_status", "moderation_status TEXT NOT NULL DEFAULT 'visible' CHECK (moderation_status IN ('visible', 'removed'))"},
		{"replies", "moderated_at", "moderated_at TEXT"},
		{"replies", "moderated_by", "moderated_by TEXT NOT NULL DEFAULT ''"},
		{"replies", "moderation_reason", "moderation_reason TEXT NOT NULL DEFAULT ''"},
	}
	for _, column := range columns {
		if err := addColumnIfMissing(tx, column.table, column.column, column.definition); err != nil {
			return err
		}
	}
	return execStatements(tx, []string{
		`CREATE TABLE content_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			target_type TEXT NOT NULL CHECK (target_type IN ('post', 'reply')),
			target_id INTEGER NOT NULL,
			reporter_user_id INTEGER NOT NULL,
			category TEXT NOT NULL CHECK (category IN ('spam', 'abuse', 'illegal', 'privacy', 'other')),
			detail TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'dismissed')),
			created_at TEXT NOT NULL,
			resolved_at TEXT,
			resolved_by TEXT NOT NULL DEFAULT '',
			resolution_note TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (event_id) REFERENCES events(id),
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (reporter_user_id) REFERENCES users(id)
		)`,
		`CREATE UNIQUE INDEX idx_content_reports_open_reporter_target
		 ON content_reports(reporter_user_id, target_type, target_id) WHERE status = 'open'`,
		`CREATE INDEX idx_content_reports_queue ON content_reports(status, created_at DESC)`,
		`CREATE INDEX idx_content_reports_target ON content_reports(target_type, target_id)`,
		`CREATE TABLE content_moderation_actions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			report_id INTEGER,
			event_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			target_type TEXT NOT NULL CHECK (target_type IN ('post', 'reply')),
			target_id INTEGER NOT NULL,
			action TEXT NOT NULL CHECK (action IN ('remove', 'restore', 'dismiss')),
			actor TEXT NOT NULL,
			reason TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (report_id) REFERENCES content_reports(id) ON DELETE SET NULL,
			FOREIGN KEY (event_id) REFERENCES events(id),
			FOREIGN KEY (post_id) REFERENCES posts(id)
		)`,
		`CREATE INDEX idx_content_moderation_actions_target
		 ON content_moderation_actions(target_type, target_id, created_at DESC)`,
	})
}

func migrateEventCoverURL(tx *sql.Tx) error {
	return addColumnIfMissing(tx, "events", "cover_url", "cover_url TEXT NOT NULL DEFAULT ''")
}

func migrateAuthenticationSessionAndPasswordReset(tx *sql.Tx) error {
	return execStatements(tx, []string{
		`CREATE TABLE user_auth_versions (
			user_id INTEGER PRIMARY KEY,
			version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
			updated_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`INSERT INTO user_auth_versions (user_id, version, updated_at)
		 SELECT id, 1, created_at FROM users`,
		`CREATE TRIGGER users_create_auth_version AFTER INSERT ON users
		 BEGIN
			INSERT INTO user_auth_versions (user_id, version, updated_at)
			VALUES (NEW.id, 1, NEW.created_at);
		 END`,
		`CREATE TABLE password_reset_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			used_at TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_password_reset_user_created
		 ON password_reset_tokens(user_id, created_at DESC)`,
		`CREATE INDEX idx_password_reset_expiry
		 ON password_reset_tokens(expires_at) WHERE used_at IS NULL`,
	})
}

func migrateAdmissionAndCheckin(tx *sql.Tx) error {
	return execStatements(tx, []string{
		`CREATE TABLE admissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			registration_id INTEGER UNIQUE,
			event_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			ticket_name TEXT NOT NULL DEFAULT '',
			credential_code TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
			issued_at TEXT NOT NULL,
			revoked_at TEXT,
			FOREIGN KEY (registration_id) REFERENCES registrations(id) ON DELETE SET NULL,
			FOREIGN KEY (event_id) REFERENCES events(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE checkins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admission_id INTEGER NOT NULL UNIQUE,
			event_id INTEGER NOT NULL,
			checked_in_at TEXT NOT NULL,
			checked_in_by TEXT NOT NULL,
			FOREIGN KEY (admission_id) REFERENCES admissions(id),
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE INDEX idx_admissions_event ON admissions(event_id)`,
		`CREATE INDEX idx_admissions_user ON admissions(user_id, issued_at DESC)`,
		`CREATE INDEX idx_checkins_event ON checkins(event_id, checked_in_at DESC)`,
		`CREATE TRIGGER checkins_immutable_update BEFORE UPDATE ON checkins
		 BEGIN SELECT RAISE(ABORT, 'checkins are immutable'); END`,
		`CREATE TRIGGER checkins_immutable_delete BEFORE DELETE ON checkins
		 BEGIN SELECT RAISE(ABORT, 'checkins are immutable'); END`,
	})
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
		{"events", "organizer_id", "organizer_id INTEGER NOT NULL DEFAULT 0 REFERENCES organizers(id)"},
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

func migrateIdentityBackfill(tx *sql.Tx) error {
	columns := []struct {
		table      string
		column     string
		definition string
	}{
		{"registrations", "identity_status", "identity_status TEXT NOT NULL DEFAULT 'legacy' CHECK (identity_status IN ('legacy', 'backfilled', 'verified'))"},
		{"posts", "identity_status", "identity_status TEXT NOT NULL DEFAULT 'legacy' CHECK (identity_status IN ('legacy', 'backfilled', 'verified'))"},
		{"replies", "identity_status", "identity_status TEXT NOT NULL DEFAULT 'legacy' CHECK (identity_status IN ('legacy', 'backfilled', 'verified'))"},
	}
	for _, column := range columns {
		if err := addColumnIfMissing(tx, column.table, column.column, column.definition); err != nil {
			return err
		}
	}

	return execStatements(tx, []string{
		`UPDATE registrations SET identity_status = 'verified' WHERE user_id IS NOT NULL AND identity_status = 'legacy'`,
		`UPDATE posts SET identity_status = 'verified' WHERE user_id IS NOT NULL AND identity_status = 'legacy'`,
		`UPDATE replies SET identity_status = 'verified' WHERE user_id IS NOT NULL AND identity_status = 'legacy'`,
		`UPDATE registrations
		 SET user_id = (SELECT users.id FROM users WHERE users.contact = registrations.contact),
		     identity_status = 'backfilled'
		 WHERE user_id IS NULL
		   AND EXISTS (SELECT 1 FROM users WHERE users.contact = registrations.contact)`,
		`UPDATE posts
		 SET user_id = (SELECT users.id FROM users WHERE users.contact = posts.author_contact),
		     identity_status = 'backfilled'
		 WHERE user_id IS NULL
		   AND EXISTS (SELECT 1 FROM users WHERE users.contact = posts.author_contact)`,
		`UPDATE replies
		 SET user_id = (SELECT users.id FROM users WHERE users.contact = replies.author_contact),
		     identity_status = 'backfilled'
		 WHERE user_id IS NULL
		   AND EXISTS (SELECT 1 FROM users WHERE users.contact = replies.author_contact)`,
		`CREATE INDEX IF NOT EXISTS idx_registrations_identity_status ON registrations(identity_status)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_identity_status ON posts(identity_status)`,
		`CREATE INDEX IF NOT EXISTS idx_replies_identity_status ON replies(identity_status)`,
	})
}

func migrateForeignKeyReadiness(tx *sql.Tx) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	if err := execStatements(tx, []string{
		fmt.Sprintf(`INSERT OR IGNORE INTO organizers
			(id, name, description, contact, logo_url, address, website, tags, created_at, updated_at)
			VALUES (0, '', '', '', '', '', '', '', '%s', '%s')`, now, now),
		`UPDATE events SET organizer_id = 0
		 WHERE NOT EXISTS (SELECT 1 FROM organizers WHERE organizers.id = events.organizer_id)`,
	}); err != nil {
		return err
	}

	hasOrganizerForeignKey, err := tableHasForeignKeyTx(tx, "events", "organizers", "organizer_id")
	if err != nil {
		return err
	}
	if !hasOrganizerForeignKey {
		if err := rebuildEventsWithForeignKey(tx); err != nil {
			return err
		}
	}

	return execStatements(tx, []string{
		`UPDATE registrations SET ticket_id = NULL
		 WHERE ticket_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM tickets WHERE tickets.id = registrations.ticket_id)`,
		`UPDATE registrations SET user_id = NULL, identity_status = 'legacy'
		 WHERE user_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users WHERE users.id = registrations.user_id)`,
		`UPDATE posts SET user_id = NULL, identity_status = 'legacy'
		 WHERE user_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users WHERE users.id = posts.user_id)`,
		`UPDATE replies SET user_id = NULL, identity_status = 'legacy'
		 WHERE user_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users WHERE users.id = replies.user_id)`,
	})
}

func rebuildEventsWithForeignKey(tx *sql.Tx) error {
	return execStatements(tx, []string{
		`CREATE TABLE events_new (
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
		`INSERT INTO events_new
			(id, organizer_id, title, description, event_time, location, capacity, price, status, created_at, updated_at)
		 SELECT id, organizer_id, title, description, event_time, location, capacity, price, status, created_at, updated_at
		 FROM events`,
		`DROP TABLE events`,
		`ALTER TABLE events_new RENAME TO events`,
		`CREATE INDEX IF NOT EXISTS idx_events_status ON events(status)`,
		`CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_events_organizer ON events(organizer_id)`,
	})
}

func validateForeignKeys(db *sql.DB) error {
	var enabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		return fmt.Errorf("读取外键状态失败: %w", err)
	}
	if enabled != 1 {
		return fmt.Errorf("SQLite 外键未启用")
	}
	required := []struct {
		table, parent, column string
	}{
		{"events", "organizers", "organizer_id"},
		{"registrations", "events", "event_id"},
		{"registrations", "tickets", "ticket_id"},
		{"registrations", "users", "user_id"},
		{"posts", "events", "event_id"},
		{"posts", "users", "user_id"},
		{"replies", "posts", "post_id"},
		{"replies", "users", "user_id"},
		{"tickets", "events", "event_id"},
		{"admissions", "registrations", "registration_id"},
		{"admissions", "events", "event_id"},
		{"admissions", "users", "user_id"},
		{"checkins", "admissions", "admission_id"},
		{"checkins", "events", "event_id"},
		{"user_auth_versions", "users", "user_id"},
		{"password_reset_tokens", "users", "user_id"},
		{"recovery_email_tokens", "users", "user_id"},
		{"content_reports", "events", "event_id"},
		{"content_reports", "posts", "post_id"},
		{"content_reports", "users", "reporter_user_id"},
		{"content_moderation_actions", "content_reports", "report_id"},
		{"content_moderation_actions", "events", "event_id"},
		{"content_moderation_actions", "posts", "post_id"},
	}
	for _, foreignKey := range required {
		exists, err := tableHasForeignKeyDB(db, foreignKey.table, foreignKey.parent, foreignKey.column)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("缺少外键约束: %s.%s -> %s", foreignKey.table, foreignKey.column, foreignKey.parent)
		}
	}

	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("执行外键一致性检查失败: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		var table string
		var rowID int64
		var parent string
		var foreignKeyID int
		if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
			return fmt.Errorf("读取外键违规记录失败: %w", err)
		}
		return fmt.Errorf("检测到外键违规: table=%s row_id=%d parent=%s fk_id=%d", table, rowID, parent, foreignKeyID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历外键检查结果失败: %w", err)
	}
	return nil
}

func tableHasForeignKeyTx(tx *sql.Tx, table, parent, column string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
	if err != nil {
		return false, fmt.Errorf("读取 %s 外键失败: %w", table, err)
	}
	defer rows.Close()
	return scanForeignKeyRows(rows, table, parent, column)
}

func tableHasForeignKeyDB(db *sql.DB, table, parent, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
	if err != nil {
		return false, fmt.Errorf("读取 %s 外键失败: %w", table, err)
	}
	defer rows.Close()
	return scanForeignKeyRows(rows, table, parent, column)
}

func scanForeignKeyRows(rows *sql.Rows, table, parent, column string) (bool, error) {
	for rows.Next() {
		var id, sequence int
		var referencedTable, fromColumn, toColumn, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &sequence, &referencedTable, &fromColumn, &toColumn, &onUpdate, &onDelete, &match); err != nil {
			return false, fmt.Errorf("解析 %s 外键失败: %w", table, err)
		}
		if referencedTable == parent && fromColumn == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("遍历 %s 外键失败: %w", table, err)
	}
	return false, nil
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
