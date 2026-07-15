package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestMigrationEmptyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s.Close()

	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
	assertNullableColumn(t, s.db, "registrations", "user_id")
	assertNullableColumn(t, s.db, "posts", "user_id")
	assertNullableColumn(t, s.db, "replies", "user_id")
	assertColumnExists(t, s.db, "registrations", "identity_status")
	assertColumnExists(t, s.db, "posts", "identity_status")
	assertColumnExists(t, s.db, "replies", "identity_status")
	assertColumnExists(t, s.db, "user_auth_versions", "version")
	assertColumnExists(t, s.db, "password_reset_tokens", "token_hash")
	assertColumnExists(t, s.db, "events", "cover_url")
	assertColumnExists(t, s.db, "posts", "moderation_status")
	assertColumnExists(t, s.db, "replies", "moderation_status")
	assertTableExists(t, s.db, "content_reports")
	assertTableExists(t, s.db, "content_moderation_actions")
	assertColumnExists(t, s.db, "users", "recovery_email")
	assertColumnExists(t, s.db, "users", "recovery_email_verified_at")
	assertTableExists(t, s.db, "recovery_email_tokens")
	assertTableExists(t, s.db, "notifications")
	assertIndexExists(t, s.db, "idx_notifications_user_created")
	assertIndexExists(t, s.db, "idx_notifications_user_unread")
	assertIndexExists(t, s.db, "idx_notifications_event")
}

func TestMigrationAddsEventCoverURLWithoutChangingExistingRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema-v7.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`,
		`CREATE TABLE organizers (id INTEGER PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', contact TEXT NOT NULL DEFAULT '', logo_url TEXT NOT NULL DEFAULT '', address TEXT NOT NULL DEFAULT '', website TEXT NOT NULL DEFAULT '', tags TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE events (id INTEGER PRIMARY KEY, organizer_id INTEGER NOT NULL DEFAULT 0, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', event_time TEXT NOT NULL, location TEXT NOT NULL, capacity INTEGER NOT NULL DEFAULT 0, price REAL NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'published', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, FOREIGN KEY (organizer_id) REFERENCES organizers(id))`,
		`CREATE TABLE tickets (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id))`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, contact TEXT NOT NULL, password_hash TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE registrations (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, user_id INTEGER, name TEXT NOT NULL, contact TEXT NOT NULL, ticket_id INTEGER, ticket_name TEXT NOT NULL DEFAULT '', identity_status TEXT NOT NULL DEFAULT 'legacy', created_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (ticket_id) REFERENCES tickets(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE posts (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, user_id INTEGER, author_name TEXT NOT NULL, author_contact TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, identity_status TEXT NOT NULL DEFAULT 'legacy', created_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE replies (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, user_id INTEGER, author_name TEXT NOT NULL, author_contact TEXT NOT NULL, content TEXT NOT NULL, identity_status TEXT NOT NULL DEFAULT 'legacy', created_at TEXT NOT NULL, FOREIGN KEY (post_id) REFERENCES posts(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE admissions (id INTEGER PRIMARY KEY, registration_id INTEGER UNIQUE, event_id INTEGER NOT NULL, user_id INTEGER NOT NULL, ticket_name TEXT NOT NULL DEFAULT '', credential_code TEXT NOT NULL UNIQUE, status TEXT NOT NULL DEFAULT 'active', issued_at TEXT NOT NULL, revoked_at TEXT, FOREIGN KEY (registration_id) REFERENCES registrations(id), FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE checkins (id INTEGER PRIMARY KEY, admission_id INTEGER NOT NULL UNIQUE, event_id INTEGER NOT NULL, checked_in_at TEXT NOT NULL, checked_in_by TEXT NOT NULL, FOREIGN KEY (admission_id) REFERENCES admissions(id), FOREIGN KEY (event_id) REFERENCES events(id))`,
		`CREATE TABLE user_auth_versions (user_id INTEGER PRIMARY KEY, version INTEGER NOT NULL, updated_at TEXT NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE password_reset_tokens (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, token_hash TEXT NOT NULL UNIQUE, expires_at TEXT NOT NULL, used_at TEXT, created_at TEXT NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id))`,
		`INSERT INTO organizers VALUES (1, '封面迁移门店', '', '', '', '', '', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO events VALUES (1, 1, '迁移前活动', '', '2099-12-31T18:00:00+08:00', '线上', 10, 0, 'published', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO users VALUES (1, '邮箱用户', 'LegacyUnique@Example.COM', 'hash', '2026-01-01T00:00:00Z')`,
		`INSERT INTO users VALUES (2, '手机用户', '13800138000', 'hash', '2026-01-02T00:00:00Z')`,
		`INSERT INTO users VALUES (3, '碰撞用户一', 'Collision@Example.COM', 'hash', '2026-01-03T00:00:00Z')`,
		`INSERT INTO users VALUES (4, '碰撞用户二', 'collision@example.com', 'hash', '2026-01-04T00:00:00Z')`,
		`INSERT INTO posts VALUES (1, 1, NULL, '历史作者', 'legacy@example.com', '历史帖子', '历史内容', 'legacy', '2026-01-01T00:00:00Z')`,
		`INSERT INTO replies VALUES (1, 1, NULL, '历史回复者', 'legacy-reply@example.com', '历史回复', 'legacy', '2026-01-01T00:00:00Z')`,
	}
	for version := 1; version <= 7; version++ {
		statements = append(statements, `INSERT INTO schema_migrations VALUES (`+fmt.Sprint(version)+`, 'applied', '2026-01-01T00:00:00Z')`)
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare v7 database: %v", err)
		}
	}
	_ = db.Close()

	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("migrate v7 database: %v", err)
	}
	defer s.Close()
	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
	assertColumnExists(t, s.db, "events", "cover_url")
	assertColumnExists(t, s.db, "posts", "moderation_status")
	assertColumnExists(t, s.db, "replies", "moderation_status")
	assertTableExists(t, s.db, "content_reports")
	assertTableExists(t, s.db, "content_moderation_actions")
	assertColumnExists(t, s.db, "users", "recovery_email")
	assertTableExists(t, s.db, "recovery_email_tokens")
	var title, coverURL string
	if err := s.db.QueryRow(`SELECT title, cover_url FROM events WHERE id = 1`).Scan(&title, &coverURL); err != nil {
		t.Fatal(err)
	}
	if title != "迁移前活动" || coverURL != "" {
		t.Fatalf("existing event changed: title=%q cover_url=%q", title, coverURL)
	}
	var postStatus, replyStatus string
	if err := s.db.QueryRow(`SELECT moderation_status FROM posts WHERE id = 1`).Scan(&postStatus); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT moderation_status FROM replies WHERE id = 1`).Scan(&replyStatus); err != nil {
		t.Fatal(err)
	}
	if postStatus != model.ModerationStatusVisible || replyStatus != model.ModerationStatusVisible {
		t.Fatalf("historical content not visible after migration: post=%q reply=%q", postStatus, replyStatus)
	}
	var recoveryEmail string
	var recoveryVerifiedAt sql.NullString
	if err := s.db.QueryRow(
		`SELECT recovery_email, recovery_email_verified_at FROM users WHERE id = 1`,
	).Scan(&recoveryEmail, &recoveryVerifiedAt); err != nil {
		t.Fatal(err)
	}
	if recoveryEmail != "legacyunique@example.com" || !recoveryVerifiedAt.Valid {
		t.Fatalf("email account recovery identity not backfilled: email=%q verified=%v", recoveryEmail, recoveryVerifiedAt.Valid)
	}
	if err := s.db.QueryRow(
		`SELECT recovery_email, recovery_email_verified_at FROM users WHERE id = 2`,
	).Scan(&recoveryEmail, &recoveryVerifiedAt); err != nil {
		t.Fatal(err)
	}
	if recoveryEmail != "" || recoveryVerifiedAt.Valid {
		t.Fatalf("phone-only account must remain unbound: email=%q verified=%v", recoveryEmail, recoveryVerifiedAt.Valid)
	}
	for _, userID := range []int64{3, 4} {
		if err := s.db.QueryRow(
			`SELECT recovery_email, recovery_email_verified_at FROM users WHERE id = ?`, userID,
		).Scan(&recoveryEmail, &recoveryVerifiedAt); err != nil {
			t.Fatal(err)
		}
		if recoveryEmail != "" || recoveryVerifiedAt.Valid {
			t.Fatalf("case-colliding account %d must remain unbound: email=%q verified=%v", userID, recoveryEmail, recoveryVerifiedAt.Valid)
		}
	}
}

func TestMigrationBackfillsMatchingIdentityAndMarksLegacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity-v3.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`,
		`INSERT INTO schema_migrations VALUES (1, 'v5_1_baseline', '2026-01-01T00:00:00Z')`,
		`INSERT INTO schema_migrations VALUES (2, 'v5_1_legacy_columns', '2026-01-01T00:00:00Z')`,
		`INSERT INTO schema_migrations VALUES (3, 'identity_user_id_expand', '2026-01-01T00:00:00Z')`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, contact TEXT NOT NULL, password_hash TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE organizers (id INTEGER PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', contact TEXT NOT NULL DEFAULT '', logo_url TEXT NOT NULL DEFAULT '', address TEXT NOT NULL DEFAULT '', website TEXT NOT NULL DEFAULT '', tags TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE events (id INTEGER PRIMARY KEY, organizer_id INTEGER NOT NULL DEFAULT 0, FOREIGN KEY (organizer_id) REFERENCES organizers(id))`,
		`CREATE TABLE tickets (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id))`,
		`CREATE TABLE registrations (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, user_id INTEGER, name TEXT NOT NULL, contact TEXT NOT NULL, ticket_id INTEGER, created_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (ticket_id) REFERENCES tickets(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE posts (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, user_id INTEGER, author_name TEXT NOT NULL, author_contact TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, created_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE replies (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, user_id INTEGER, author_name TEXT NOT NULL, author_contact TEXT NOT NULL, content TEXT NOT NULL, created_at TEXT NOT NULL, FOREIGN KEY (post_id) REFERENCES posts(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`INSERT INTO events VALUES (10, 0)`,
		`INSERT INTO users VALUES (7, '匹配用户', 'match@example.com', 'hash', '2026-01-01T00:00:00Z')`,
		`INSERT INTO registrations VALUES (1, 10, NULL, '匹配用户', 'match@example.com', NULL, '2026-01-01T00:00:00Z')`,
		`INSERT INTO registrations VALUES (2, 10, NULL, '遗留用户', 'legacy@example.com', NULL, '2026-01-01T00:00:00Z')`,
		`INSERT INTO posts VALUES (1, 10, NULL, '匹配用户', 'match@example.com', '标题', '内容', '2026-01-01T00:00:00Z')`,
		`INSERT INTO replies VALUES (1, 1, NULL, '遗留用户', 'legacy@example.com', '回复', '2026-01-01T00:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare v3 database: %v", err)
		}
	}
	_ = db.Close()

	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s.Close()

	var userID sql.NullInt64
	var status string
	if err := s.db.QueryRow(`SELECT user_id, identity_status FROM registrations WHERE id = 1`).Scan(&userID, &status); err != nil {
		t.Fatal(err)
	}
	if !userID.Valid || userID.Int64 != 7 || status != "backfilled" {
		t.Fatalf("unexpected backfill: user_id=%v status=%s", userID, status)
	}
	if err := s.db.QueryRow(`SELECT user_id, identity_status FROM registrations WHERE id = 2`).Scan(&userID, &status); err != nil {
		t.Fatal(err)
	}
	if userID.Valid || status != "legacy" {
		t.Fatalf("unexpected legacy record: user_id=%v status=%s", userID, status)
	}

	report, err := s.GetIdentityMigrationReport(100)
	if err != nil {
		t.Fatal(err)
	}
	if report.Registrations.Total != 2 || report.Registrations.Backfilled != 1 || report.Registrations.Legacy != 1 {
		t.Fatalf("unexpected registration report: %+v", report.Registrations)
	}
	if len(report.LegacyRecords) != 2 {
		t.Fatalf("expected registration and reply legacy records, got %d", len(report.LegacyRecords))
	}
}

func TestDatabaseBackupRestoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	backupPath := filepath.Join(dir, "app.db.bak")

	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOrganizer(&model.Organizer{Name: "备份门店"}); err != nil {
		t.Fatal(err)
	}
	event := &model.Event{OrganizerID: 1, Title: "备份活动", EventTime: "2099-12-31T18:00:00+08:00", Location: "线上", Capacity: 10}
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	s, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEvent(event.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, backup, 0o600); err != nil {
		t.Fatal(err)
	}

	s, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	restored, err := s.GetEvent(event.ID)
	if err != nil || restored == nil || restored.Title != event.Title {
		t.Fatalf("restored event mismatch: event=%+v err=%v", restored, err)
	}
	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
}

func TestMigrationRebuildsEventsWhenOrganizerForeignKeyIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-event-fk.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`,
		`INSERT INTO schema_migrations VALUES (1, 'v5_1_baseline', '2026-01-01T00:00:00Z')`,
		`INSERT INTO schema_migrations VALUES (2, 'v5_1_legacy_columns', '2026-01-01T00:00:00Z')`,
		`INSERT INTO schema_migrations VALUES (3, 'identity_user_id_expand', '2026-01-01T00:00:00Z')`,
		`INSERT INTO schema_migrations VALUES (4, 'identity_backfill_and_legacy_status', '2026-01-01T00:00:00Z')`,
		`CREATE TABLE organizers (id INTEGER PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', contact TEXT NOT NULL DEFAULT '', logo_url TEXT NOT NULL DEFAULT '', address TEXT NOT NULL DEFAULT '', website TEXT NOT NULL DEFAULT '', tags TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, contact TEXT NOT NULL, password_hash TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, organizer_id INTEGER NOT NULL DEFAULT 0, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', event_time TEXT NOT NULL, location TEXT NOT NULL, capacity INTEGER NOT NULL DEFAULT 0, price REAL NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'published', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE tickets (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, name TEXT NOT NULL, price REAL NOT NULL DEFAULT 0, stock INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id))`,
		`CREATE TABLE registrations (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, user_id INTEGER, name TEXT NOT NULL, contact TEXT NOT NULL, ticket_id INTEGER, ticket_name TEXT NOT NULL DEFAULT '', identity_status TEXT NOT NULL DEFAULT 'legacy', created_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (ticket_id) REFERENCES tickets(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE posts (id INTEGER PRIMARY KEY, event_id INTEGER NOT NULL, user_id INTEGER, author_name TEXT NOT NULL, author_contact TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, identity_status TEXT NOT NULL DEFAULT 'legacy', created_at TEXT NOT NULL, FOREIGN KEY (event_id) REFERENCES events(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`CREATE TABLE replies (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, user_id INTEGER, author_name TEXT NOT NULL, author_contact TEXT NOT NULL, content TEXT NOT NULL, identity_status TEXT NOT NULL DEFAULT 'legacy', created_at TEXT NOT NULL, FOREIGN KEY (post_id) REFERENCES posts(id), FOREIGN KEY (user_id) REFERENCES users(id))`,
		`INSERT INTO organizers VALUES (1, '原门店', '', '', '', '', '', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO events VALUES (1, 1, '保留活动', '', '2099-01-01T00:00:00Z', '线上', 10, 0, 'published', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare missing-fk database: %v", err)
		}
	}
	_ = db.Close()

	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s.Close()
	hasForeignKey, err := tableHasForeignKeyDB(s.db, "events", "organizers", "organizer_id")
	if err != nil || !hasForeignKey {
		t.Fatalf("organizer foreign key not rebuilt: exists=%v err=%v", hasForeignKey, err)
	}
	event, err := s.GetEvent(1)
	if err != nil || event == nil || event.Title != "保留活动" {
		t.Fatalf("event not preserved after rebuild: event=%+v err=%v", event, err)
	}
}

func TestMigrationLegacyDatabasePreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	legacyStatements := []string{
		`CREATE TABLE events (
			id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '', event_time TEXT NOT NULL,
			location TEXT NOT NULL, capacity INTEGER NOT NULL DEFAULT 0,
			price REAL NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'published',
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE registrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT, event_id INTEGER NOT NULL,
			name TEXT NOT NULL, contact TEXT NOT NULL, created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT, event_id INTEGER NOT NULL,
			author_name TEXT NOT NULL, author_contact TEXT NOT NULL,
			title TEXT NOT NULL, content TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE TABLE replies (
			id INTEGER PRIMARY KEY AUTOINCREMENT, post_id INTEGER NOT NULL,
			author_name TEXT NOT NULL, author_contact TEXT NOT NULL,
			content TEXT NOT NULL, created_at TEXT NOT NULL,
			FOREIGN KEY (post_id) REFERENCES posts(id)
		)`,
		`INSERT INTO events (title, event_time, location, capacity, created_at, updated_at)
		 VALUES ('旧活动', '2026-12-31T18:00:00+08:00', '线上', 10, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO registrations (event_id, name, contact, created_at)
		 VALUES (1, '旧用户', 'legacy@example.com', '2026-01-01T00:00:00Z')`,
		`INSERT INTO posts (event_id, author_name, author_contact, title, content, created_at)
		 VALUES (1, '旧用户', 'legacy@example.com', '旧帖子', '内容', '2026-01-01T00:00:00Z')`,
		`INSERT INTO replies (post_id, author_name, author_contact, content, created_at)
		 VALUES (1, '旧用户', 'legacy@example.com', '回复', '2026-01-01T00:00:00Z')`,
	}
	for _, statement := range legacyStatements {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare legacy database: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}
	defer s.Close()

	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
	assertNullableColumn(t, s.db, "registrations", "user_id")
	assertNullableColumn(t, s.db, "posts", "user_id")
	assertNullableColumn(t, s.db, "replies", "user_id")
	assertColumnExists(t, s.db, "registrations", "ticket_id")
	assertColumnExists(t, s.db, "registrations", "ticket_name")
	assertColumnExists(t, s.db, "events", "organizer_id")

	for table, want := range map[string]int{"events": 1, "registrations": 1, "posts": 1, "replies": 1} {
		var got int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("expected %d rows in %s, got %d", want, table, got)
		}
	}
}

func TestMigrationV5ToCurrentPreservesRegistrationsWithoutSyntheticAdmissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v5-to-v6.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatalf("create current database: %v", err)
	}
	statements := []string{
		`INSERT INTO organizers (id, name, created_at, updated_at) VALUES (1, '迁移门店', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO users (id, name, contact, password_hash, created_at) VALUES (7, '迁移用户', 'migration@example.com', 'hash', '2026-01-01T00:00:00Z')`,
		`INSERT INTO events (id, organizer_id, title, event_time, location, capacity, created_at, updated_at) VALUES (10, 1, '迁移活动', '2099-01-01T00:00:00Z', '线上', 10, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO registrations (id, event_id, user_id, name, contact, identity_status, created_at) VALUES (20, 10, 7, '迁移用户', 'migration@example.com', 'verified', '2026-01-01T00:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			_ = s.Close()
			t.Fatalf("seed current database: %v", err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close current database: %v", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open database for v5 fixture: %v", err)
	}
	v6Objects := []string{
		`DROP TRIGGER users_create_auth_version`,
		`DROP TABLE password_reset_tokens`,
		`DROP TABLE user_auth_versions`,
		`DELETE FROM schema_migrations WHERE version = 7`,
		`DROP TRIGGER checkins_immutable_update`,
		`DROP TRIGGER checkins_immutable_delete`,
		`DROP TABLE checkins`,
		`DROP TABLE admissions`,
		`DELETE FROM schema_migrations WHERE version = 6`,
	}
	for _, statement := range v6Objects {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare v5 fixture: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close v5 fixture: %v", err)
	}

	s, err = OpenStore(path)
	if err != nil {
		t.Fatalf("migrate v5 database: %v", err)
	}
	defer s.Close()

	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
	var eventID, userID int64
	var name, contact string
	if err := s.db.QueryRow(
		`SELECT event_id, user_id, name, contact FROM registrations WHERE id = 20`,
	).Scan(&eventID, &userID, &name, &contact); err != nil {
		t.Fatalf("read preserved registration: %v", err)
	}
	if eventID != 10 || userID != 7 || name != "迁移用户" || contact != "migration@example.com" {
		t.Fatalf("registration changed during migration: event=%d user=%d name=%q contact=%q", eventID, userID, name, contact)
	}
	var admissionCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM admissions`).Scan(&admissionCount); err != nil {
		t.Fatalf("count admissions: %v", err)
	}
	if admissionCount != 0 {
		t.Fatalf("expected no synthetic admissions for existing registrations, got %d", admissionCount)
	}
}

func TestMigrationV6ToV7PreservesUsersAndInitializesAuthVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v6-to-v7.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "迁移认证用户", Contact: "auth-migration@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`DROP TRIGGER users_create_auth_version`,
		`DROP TABLE password_reset_tokens`,
		`DROP TABLE user_auth_versions`,
		`DELETE FROM schema_migrations WHERE version = 7`,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare v6 fixture: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = OpenStore(path)
	if err != nil {
		t.Fatalf("migrate v6 database: %v", err)
	}
	defer s.Close()
	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
	preserved, err := s.GetUserByContact(user.Contact)
	if err != nil || preserved == nil || preserved.ID != user.ID || preserved.AuthVersion != 1 {
		t.Fatalf("user/auth version not preserved: user=%+v err=%v", preserved, err)
	}
	var tokenCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens`).Scan(&tokenCount); err != nil || tokenCount != 0 {
		t.Fatalf("unexpected reset token state: count=%d err=%v", tokenCount, err)
	}
}

func TestMigrationV10ToV11AddsNotificationsWithoutChangingExistingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v10-to-v11.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	organizer := &model.Organizer{Name: "通知迁移门店"}
	if err := s.CreateOrganizer(organizer); err != nil {
		t.Fatal(err)
	}
	event := &model.Event{
		OrganizerID: organizer.ID, Title: "迁移前活动", EventTime: "2099-01-01T00:00:00Z",
		Location: "线上", Capacity: 10,
	}
	if err := s.CreateEvent(event); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Name: "迁移用户", Contact: "notification-migration@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`DROP INDEX idx_notifications_event`,
		`DROP INDEX idx_notifications_user_unread`,
		`DROP INDEX idx_notifications_user_created`,
		`DROP TABLE notifications`,
		`DELETE FROM schema_migrations WHERE version = 11`,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatalf("prepare v10 fixture: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = OpenStore(path)
	if err != nil {
		t.Fatalf("migrate v10 database: %v", err)
	}
	defer s.Close()
	assertSchemaVersion(t, s.db, CurrentSchemaVersion)
	assertTableExists(t, s.db, "notifications")
	assertIndexExists(t, s.db, "idx_notifications_user_created")
	assertIndexExists(t, s.db, "idx_notifications_user_unread")
	assertIndexExists(t, s.db, "idx_notifications_event")
	if migrated, err := s.GetEvent(event.ID); err != nil || migrated == nil || migrated.Title != event.Title {
		t.Fatalf("event changed during notification migration: event=%+v err=%v", migrated, err)
	}
	if migrated, err := s.GetUserByContact(user.Contact); err != nil || migrated == nil || migrated.ID != user.ID {
		t.Fatalf("user changed during notification migration: user=%+v err=%v", migrated, err)
	}
	if has, err := tableHasForeignKeyDB(s.db, "notifications", "users", "user_id"); err != nil || !has {
		t.Fatalf("notification user foreign key missing: exists=%v err=%v", has, err)
	}
	if has, err := tableHasForeignKeyDB(s.db, "notifications", "events", "event_id"); err != nil || !has {
		t.Fatalf("notification event foreign key missing: exists=%v err=%v", has, err)
	}
}

func TestMigrationRepeatedExecution(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repeat.db")
	for i := 0; i < 2; i++ {
		s, err := OpenStore(path)
		if err != nil {
			t.Fatalf("OpenStore pass %d: %v", i+1, err)
		}
		assertSchemaVersion(t, s.db, CurrentSchemaVersion)
		var count int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
			t.Fatalf("count migrations: %v", err)
		}
		if count != CurrentSchemaVersion {
			t.Fatalf("expected %d migration rows, got %d", CurrentSchemaVersion, count)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("close pass %d: %v", i+1, err)
		}
	}
}

func TestMigrationFailureIsReturned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open broken database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("prepare broken migration table: %v", err)
	}
	_ = db.Close()

	_, err = OpenStore(path)
	if err == nil {
		t.Fatal("expected migration error")
	}
	if !strings.Contains(err.Error(), "记录迁移") {
		t.Fatalf("expected migration record error, got: %v", err)
	}
}

func assertSchemaVersion(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&got); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if got != want {
		t.Fatalf("expected schema version %d, got %d", want, got)
	}
}

func assertNullableColumn(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	notNull, found := columnNotNull(t, db, table, column)
	if !found {
		t.Fatalf("expected %s.%s column", table, column)
	}
	if notNull != 0 {
		t.Fatalf("expected %s.%s to be nullable", table, column)
	}
}

func assertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	_, found := columnNotNull(t, db, table, column)
	if !found {
		t.Fatalf("expected %s.%s column", table, column)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		t.Fatalf("query table %s: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("expected table %s", table)
	}
}

func assertIndexExists(t *testing.T, db *sql.DB, index string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&count); err != nil {
		t.Fatalf("query index %s: %v", index, err)
	}
	if count != 1 {
		t.Fatalf("expected index %s", index)
	}
}

func columnNotNull(t *testing.T, db *sql.DB, table, column string) (int, bool) {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("table info %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan table info %s: %v", table, err)
		}
		if name == column {
			return notNull, true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info %s: %v", table, err)
	}
	return 0, false
}
