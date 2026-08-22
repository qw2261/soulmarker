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

const CurrentSchemaVersion = 15

type Store struct {
	db             *sql.DB
	registrationMu sync.Mutex
	checkinMu      sync.Mutex
	moderationMu   sync.Mutex
	notificationMu sync.Mutex
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

// SchemaVersion 返回数据库当前已应用的 schema_migrations 最高版本。
// 供 readiness 探针校验数据库是否已迁移到应用期望的 CurrentSchemaVersion。
func (s *Store) SchemaVersion() (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}
	var version int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("读取迁移版本失败: %w", err)
	}
	return version, nil
}

// Backup 将当前数据库的一致性快照写入 destPath，返回写入的文件大小。
// 使用 VACUUM INTO，目标文件将被完整重建并生成一份经过 checkpoint 的紧凑快照，
// 可在应用运行期间安全执行，且不会写入任何业务状态。
func (s *Store) Backup(destPath string) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}
	if _, err := s.db.Exec(`VACUUM INTO ?`, destPath); err != nil {
		return 0, fmt.Errorf("创建数据库备份失败: %w", err)
	}
	info, err := os.Stat(destPath)
	if err != nil {
		return 0, fmt.Errorf("读取备份文件信息失败: %w", err)
	}
	return info.Size(), nil
}

// prepareDB 打开 SQLite 数据库并应用连接级基础配置（WAL / busy_timeout）。
// 供 OpenStore 与独立迁移入口 Migrate 共用，保证两者对库的初始化一致。
func prepareDB(dbPath string) (*sql.DB, error) {
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
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("设置 WAL 模式失败: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("设置 busy_timeout 失败: %w", err)
	}
	return db, nil
}

// OpenStore 打开数据库并执行版本化迁移，任何初始化失败都会返回给调用方。
func OpenStore(dbPath string) (*Store, error) {
	db, err := prepareDB(dbPath)
	if err != nil {
		return nil, err
	}
	closeOnError := func(err error) (*Store, error) {
		_ = db.Close()
		return nil, err
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

// Migrate 独立执行数据库迁移，供 `event-go migrate` 命令在应用灰度前单独运行，
// 使数据库先于新应用版本就绪。与 OpenStore 使用同一批迁移（migrations()），
// 将库推进到当前 Schema（CurrentSchemaVersion）并完成外键一致性校验，随后关闭数据库。
// 幂等：已应用过的迁移不会重复执行；返回最终 Schema 版本。
func Migrate(dbPath string) (int, error) {
	db, err := prepareDB(dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	if err := migrate(db); err != nil {
		return 0, fmt.Errorf("数据库迁移失败: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return 0, fmt.Errorf("启用外键失败: %w", err)
	}
	if err := validateForeignKeys(db); err != nil {
		return 0, err
	}
	var version int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("读取迁移版本失败: %w", err)
	}
	return version, nil
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
		{version: 11, name: "in_app_notifications", apply: migrateInAppNotifications},
		{version: 12, name: "organization_tenant_foundation", apply: migrateOrganizationTenantFoundation},
		{version: 13, name: "event_tenant_scope", apply: migrateEventTenantScope},
		{version: 14, name: "organization_audit_log", apply: migrateOrganizationAuditLog},
		{version: 15, name: "data_subject_privacy", apply: migrateDataSubjectPrivacy},
	}
}

func migrateOrganizationAuditLog(tx *sql.Tx) error {
	return execStatements(tx, []string{
		`CREATE TABLE organization_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL REFERENCES organizations(id),
			actor_type TEXT NOT NULL
			 CHECK (actor_type IN ('organization_member', 'platform_admin', 'system')),
			actor_id INTEGER,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL,
			outcome TEXT NOT NULL
			 CHECK (outcome IN ('success', 'denied', 'failure')),
			http_status INTEGER NOT NULL CHECK (http_status BETWEEN 100 AND 599),
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX idx_organization_audit_org_created
		 ON organization_audit_logs(organization_id, created_at DESC, id DESC)`,
		`CREATE INDEX idx_organization_audit_request
		 ON organization_audit_logs(request_id)`,
		`CREATE INDEX idx_organization_audit_action
		 ON organization_audit_logs(organization_id, action, created_at DESC)`,
		`CREATE TRIGGER organization_audit_logs_immutable_update
		 BEFORE UPDATE ON organization_audit_logs
		 BEGIN
			SELECT RAISE(ABORT, 'organization audit logs are immutable');
		 END`,
		`CREATE TRIGGER organization_audit_logs_immutable_delete
		 BEFORE DELETE ON organization_audit_logs
		 BEGIN
			SELECT RAISE(ABORT, 'organization audit logs are immutable');
		 END`,
	})
}

// migrateDataSubjectPrivacy 为数据主体（隐私）权添加支撑：users.deleted_at 标记注销时间，
// data_subject_requests 记录账号注销与数据导出等请求及其处置状态，满足隐私请求/留存流程的可追溯性。
func migrateDataSubjectPrivacy(tx *sql.Tx) error {
	if err := addColumnIfMissing(tx, "users", "deleted_at", "deleted_at TEXT"); err != nil {
		return err
	}
	return execStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS data_subject_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			request_type TEXT NOT NULL
			 CHECK (request_type IN ('account_erasure', 'data_export')),
			status TEXT NOT NULL
			 CHECK (status IN ('pending', 'completed', 'rejected', 'failed')),
			requested_at TEXT NOT NULL,
			processed_at TEXT,
			processed_by INTEGER,
			resolution TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX idx_data_subject_requests_user
		 ON data_subject_requests(user_id, requested_at DESC, id DESC)`,
	})
}

func migrateEventTenantScope(tx *sql.Tx) error {
	if err := addColumnIfMissing(
		tx, "events", "organization_id",
		"organization_id INTEGER NOT NULL DEFAULT 0 REFERENCES organizations(id)",
	); err != nil {
		return err
	}
	return execStatements(tx, []string{
		`UPDATE events
		 SET organization_id = COALESCE((
			SELECT organization_id FROM organizers WHERE organizers.id = events.organizer_id
		 ), 0)`,
		`CREATE INDEX idx_events_organization
		 ON events(organization_id, id DESC)`,
		`CREATE TRIGGER events_reject_tenant_mismatch_insert
		 BEFORE INSERT ON events
		 WHEN NEW.organizer_id <> 0 AND NEW.organization_id <> 0
		  AND NOT EXISTS (
			SELECT 1 FROM organizers
			WHERE id = NEW.organizer_id AND organization_id = NEW.organization_id
		  )
		 BEGIN
			SELECT RAISE(ABORT, 'event organization mismatch');
		 END`,
		`CREATE TRIGGER events_reject_tenant_mismatch_update
		 BEFORE UPDATE OF organizer_id, organization_id ON events
		 WHEN NEW.organization_id <> OLD.organization_id
		  AND NEW.organizer_id <> 0 AND NEW.organization_id <> 0
		  AND NOT EXISTS (
			SELECT 1 FROM organizers
			WHERE id = NEW.organizer_id AND organization_id = NEW.organization_id
		  )
		 BEGIN
			SELECT RAISE(ABORT, 'event organization mismatch');
		 END`,
		`CREATE TRIGGER events_assign_tenant_after_insert
		 AFTER INSERT ON events
		 WHEN NEW.organizer_id <> 0 AND NEW.organization_id = 0
		 BEGIN
			UPDATE events
			SET organization_id = (
				SELECT organization_id FROM organizers WHERE id = NEW.organizer_id
			)
			WHERE id = NEW.id;
		 END`,
		`CREATE TRIGGER events_sync_tenant_after_organizer_update
		 AFTER UPDATE OF organizer_id ON events
		 WHEN NEW.organizer_id <> 0 AND NEW.organizer_id <> OLD.organizer_id
		 BEGIN
			UPDATE events
			SET organization_id = (
				SELECT organization_id FROM organizers WHERE id = NEW.organizer_id
			)
			WHERE id = NEW.id;
		 END`,
	})
}

func migrateOrganizationTenantFoundation(tx *sql.Tx) error {
	if err := execStatements(tx, []string{
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'unclaimed'
			 CHECK (status IN ('unclaimed', 'active', 'suspended', 'system')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`INSERT INTO organizations (id, name, slug, status, created_at, updated_at)
		 SELECT id, name,
			CASE WHEN id = 0 THEN 'system' ELSE 'legacy-' || id END,
			CASE WHEN id = 0 THEN 'system' ELSE 'unclaimed' END,
			created_at, updated_at
		 FROM organizers`,
		`CREATE UNIQUE INDEX idx_organizations_slug
		 ON organizations(slug) WHERE slug <> ''`,
		`CREATE INDEX idx_organizations_status_created
		 ON organizations(status, created_at DESC)`,
	}); err != nil {
		return err
	}
	if err := addColumnIfMissing(
		tx, "organizers", "organization_id",
		"organization_id INTEGER NOT NULL DEFAULT 0 REFERENCES organizations(id)",
	); err != nil {
		return err
	}
	return execStatements(tx, []string{
		`UPDATE organizers SET organization_id = id`,
		`CREATE UNIQUE INDEX idx_organizers_organization
		 ON organizers(organization_id) WHERE organization_id <> 0`,
		`CREATE TRIGGER organizers_create_unclaimed_organization
		 AFTER INSERT ON organizers
		 WHEN NEW.id <> 0 AND NEW.organization_id = 0
		 BEGIN
			INSERT INTO organizations (name, slug, status, created_at, updated_at)
			VALUES (NEW.name, '', 'unclaimed', NEW.created_at, NEW.updated_at);
			UPDATE organizers SET organization_id = last_insert_rowid() WHERE id = NEW.id;
		 END`,
		`CREATE TRIGGER organizers_suspend_organization_after_delete
		 AFTER DELETE ON organizers
		 WHEN OLD.organization_id <> 0
		 BEGIN
			UPDATE organizations
			SET status = 'suspended', updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
			WHERE id = OLD.organization_id;
		 END`,
		`CREATE TABLE organization_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL
			 CHECK (role IN ('owner', 'admin', 'editor', 'checker', 'finance')),
			status TEXT NOT NULL DEFAULT 'active'
			 CHECK (status IN ('active', 'revoked')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE (organization_id, user_id)
		)`,
		`CREATE INDEX idx_organization_members_user
		 ON organization_members(user_id, status, created_at ASC)`,
		`CREATE INDEX idx_organization_members_org_role
		 ON organization_members(organization_id, status, role)`,
		`CREATE UNIQUE INDEX idx_organization_members_active_owner
		 ON organization_members(organization_id) WHERE role = 'owner' AND status = 'active'`,
		`CREATE TABLE organization_invitations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			role TEXT NOT NULL
			 CHECK (role IN ('admin', 'editor', 'checker', 'finance')),
			token_hash TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'pending'
			 CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
			expires_at TEXT NOT NULL,
			accepted_at TEXT,
			revoked_at TEXT,
			invited_by_user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
			FOREIGN KEY (invited_by_user_id) REFERENCES users(id)
		)`,
		`CREATE UNIQUE INDEX idx_organization_invitations_pending
		 ON organization_invitations(organization_id, email) WHERE status = 'pending'`,
		`CREATE INDEX idx_organization_invitations_expiry
		 ON organization_invitations(expires_at) WHERE status = 'pending'`,
	})
}

func migrateInAppNotifications(tx *sql.Tx) error {
	return execStatements(tx, []string{
		`CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			event_id INTEGER,
			type TEXT NOT NULL CHECK (type IN ('registration_confirmed', 'registration_cancelled', 'event_updated', 'event_reminder_24h')),
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			action_url TEXT NOT NULL DEFAULT '',
			idempotency_key TEXT NOT NULL UNIQUE,
			read_at TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC, id DESC)`,
		`CREATE INDEX idx_notifications_user_unread ON notifications(user_id, read_at, created_at DESC)`,
		`CREATE INDEX idx_notifications_event ON notifications(event_id)`,
	})
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
		{"events", "organizations", "organization_id"},
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
		{"notifications", "users", "user_id"},
		{"notifications", "events", "event_id"},
		{"organizers", "organizations", "organization_id"},
		{"organization_members", "organizations", "organization_id"},
		{"organization_members", "users", "user_id"},
		{"organization_invitations", "organizations", "organization_id"},
		{"organization_invitations", "users", "invited_by_user_id"},
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
