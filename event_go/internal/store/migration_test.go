package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
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
			name TEXT NOT NULL, contact TEXT NOT NULL, created_at TEXT NOT NULL
		)`,
		`CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT, event_id INTEGER NOT NULL,
			author_name TEXT NOT NULL, author_contact TEXT NOT NULL,
			title TEXT NOT NULL, content TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
		)`,
		`CREATE TABLE replies (
			id INTEGER PRIMARY KEY AUTOINCREMENT, post_id INTEGER NOT NULL,
			author_name TEXT NOT NULL, author_contact TEXT NOT NULL,
			content TEXT NOT NULL, created_at TEXT NOT NULL
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
