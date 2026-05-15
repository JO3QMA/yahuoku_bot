package sqlite

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestOpen_success(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
}

func TestOpen_mkdirError(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "sub/x.db"), WithMkdirAll(func(string, os.FileMode) error {
		return errors.New("mkdir")
	}))
	if err == nil || err.Error() != "create db directory: mkdir" {
		t.Fatalf("err=%v", err)
	}
}

func TestOpen_absErrorIgnored(t *testing.T) {
	dir := t.TempDir()
	_, err := Open(filepath.Join(dir, "a.db"), WithFilepathAbs(func(string) (string, error) {
		return "", errors.New("abs")
	}))
	if err != nil {
		t.Fatalf("should still open: %v", err)
	}
}

func TestOpen_WALFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.db")
	var n int
	db, err := Open(path, WithSQLOpen(func(driver, dsn string) (*sql.DB, error) {
		n++
		if n == 1 {
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectExec("PRAGMA journal_mode=WAL").WillReturnError(errors.New("wal fail"))
			return mockDB, nil
		}
		return sql.Open(driver, dsn)
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
}

func TestOpen_DELETEJournal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.db")
	t.Setenv("SQLITE_JOURNAL_MODE", "DELETE")
	t.Cleanup(func() { _ = os.Unsetenv("SQLITE_JOURNAL_MODE") })

	db, err := Open(path, WithEnv(func(k string) string {
		if k == "SQLITE_JOURNAL_MODE" {
			return "DELETE"
		}
		return os.Getenv(k)
	}))
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
}

func TestOpen_mmapError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("PRAGMA mmap_size=0").WillReturnError(errors.New("mmap"))
	_, err = Open("/tmp/unused", WithSQLOpen(func(string, string) (*sql.DB, error) {
		return mockDB, nil
	}), WithEnv(func(string) string { return "DELETE" }))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_journalDeleteError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("PRAGMA mmap_size=0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("PRAGMA journal_mode=DELETE").WillReturnError(errors.New("del"))
	_, err = Open("/x", WithSQLOpen(func(string, string) (*sql.DB, error) {
		return mockDB, nil
	}), WithEnv(func(string) string { return "DELETE" }))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_sqlOpenError(t *testing.T) {
	_, err := Open("x.db", WithSQLOpen(func(string, string) (*sql.DB, error) {
		return nil, errors.New("open")
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_secondOpenFails(t *testing.T) {
	var n int
	_, err := Open("p.db", WithSQLOpen(func(driver, dsn string) (*sql.DB, error) {
		n++
		if n == 1 {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectExec("PRAGMA journal_mode=WAL").WillReturnError(errors.New("wal"))
			return db, nil
		}
		return nil, errors.New("second")
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_schemaError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("PRAGMA journal_mode=WAL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE").WillReturnError(errors.New("schema"))
	_, err = Open("x", WithSQLOpen(func(string, string) (*sql.DB, error) {
		return mockDB, nil
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_withRemoveOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rm.db")
	_, err := Open(path, WithRemove(func(string) error { return nil }))
	if err != nil {
		t.Fatal(err)
	}
}
