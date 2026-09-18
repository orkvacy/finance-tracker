// Package db membuka SQLite dan menjalankan migrasi.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB memegang dua handle ke berkas yang sama.
//
// SQLite hanya mengizinkan SATU penulis pada satu waktu. Dengan satu *sql.DB
// berpool default, dua penulis bersamaan menghasilkan "database is locked" —
// dan galat itu hampir selalu baru muncul di produksi, bukan saat development.
// Memisahkan handle tulis (maks 1 koneksi) dari handle baca menghilangkan
// seluruh kelas bug itu tanpa memperlambat apa pun: mode WAL mengizinkan
// pembacaan berjalan paralel dengan penulisan.
type DB struct {
	W *sql.DB // penulis — SetMaxOpenConns(1)
	R *sql.DB // pembaca
}

func dsn(path string) string {
	p := url.Values{}
	p.Add("_pragma", "journal_mode(WAL)")
	p.Add("_pragma", "synchronous(NORMAL)") // aman dipasangkan dengan WAL
	p.Add("_pragma", "busy_timeout(5000)")
	p.Add("_pragma", "foreign_keys(ON)")
	p.Add("_pragma", "cache_size(-8000)") // 8 MB, sejalan dengan NFR-10
	return "file:" + filepath.ToSlash(path) + "?" + p.Encode()
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("membuat direktori database: %w", err)
	}

	w, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("membuka handle tulis: %w", err)
	}
	w.SetMaxOpenConns(1)

	r, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		w.Close()
		return nil, fmt.Errorf("membuka handle baca: %w", err)
	}
	r.SetMaxOpenConns(4)

	if err := w.Ping(); err != nil {
		w.Close()
		r.Close()
		return nil, fmt.Errorf("menghubungi database: %w", err)
	}
	return &DB{W: w, R: r}, nil
}

func (d *DB) Close() error {
	d.R.Close()
	return d.W.Close()
}

// Migrate menerapkan berkas .sql yang belum pernah dijalankan, berurutan nama.
// Idempoten: aman dipanggil di setiap startup, sesuai §13.5.
func (d *DB) Migrate(ctx context.Context) error {
	if _, err := d.W.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
		   name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("membuat tabel migrasi: %w", err)
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var seen int
		if err := d.W.QueryRowContext(ctx,
			`SELECT count(*) FROM schema_migrations WHERE name = ?`, name).Scan(&seen); err != nil {
			return err
		}
		if seen > 0 {
			continue
		}

		body, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}

		tx, err := d.W.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrasi %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (name, applied_at) VALUES (?, datetime('now'))`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migrasi %s: %w", name, err)
		}
	}
	return nil
}
