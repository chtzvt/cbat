package store

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore is a persistent file store backed by SQLite.
// It only handles the Files namespace — Variables and Labels
// should be kept in a separate MemoryStore per VM instance.
type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex // serializes writes; reads can be concurrent
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Get(ns Namespace, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var val string
	err := s.db.QueryRow("SELECT value FROM files WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", false
	}
	return val, true
}

func (s *SQLiteStore) Set(ns Namespace, key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec("INSERT OR REPLACE INTO files (key, value) VALUES (?, ?)", key, value)
}

func (s *SQLiteStore) Delete(ns Namespace, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec("DELETE FROM files WHERE key = ?", key)
}

func (s *SQLiteStore) Exists(ns Namespace, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM files WHERE key = ?", key).Scan(&count)
	return count > 0
}

func (s *SQLiteStore) Keys(ns Namespace) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT key FROM files")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		rows.Scan(&k)
		keys = append(keys, k)
	}
	return keys
}

func (s *SQLiteStore) Snapshot(ns Namespace) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT key, value FROM files")
	if err != nil {
		return nil
	}
	defer rows.Close()
	snap := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		snap[k] = v
	}
	return snap
}
