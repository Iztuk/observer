package audit

import (
	"cf-observer/internal/config"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

var DatabaseStore SQLiteStore

type Findings struct {
	ID        string
	JobID     string
	RuleID    string
	Title     string
	Message   string
	CreatedAt time.Time
}

type Store interface {
	SaveJob(job Job) error
}

func (s *SQLiteStore) SaveJob(job Job) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO audit_jobs (
	id, 
	type, 
	request_id,
	host, 
	method, 
	path, 
	query, 
	upstream, 
	status, 
	timestamp, 
	duration_ms, 
	headers, 
	body, 
	error)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	var (
		id      string
		jobType string
		meta    Metadata
		headers string
		body    string
		errStr  string
	)

	switch j := job.(type) {
	case *RequestJob:
		id = newUUID()
		jobType = string(j.JobType())
		meta = j.Meta

		headers, err = marshalHeaders(j.Headers)
		if err != nil {
			return err
		}

		body = string(j.Body)
	case *ResponseJob:
		id = newUUID()
		jobType = string(j.JobType())
		meta = j.Meta

		headers, err = marshalHeaders(j.Headers)
		if err != nil {
			return err
		}

		body = string(j.Body)
	case *FailureJob:
		id = newUUID()
		jobType = string(j.JobType())
		meta = j.Meta

		errStr = j.Error
	}

	_, err = tx.Exec(
		query,
		id,
		jobType,
		meta.RequestID,
		meta.Host,
		meta.Method,
		meta.Path,
		meta.Query,
		meta.Upstream,
		meta.Status,
		meta.Timestamp.Format(time.RFC3339Nano),
		meta.DurationMs,
		headers,
		body,
		errStr,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) Connect(logger *log.Logger) error {
	configDir, err := config.ConfigDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(configDir, "cf-observer.db")

	s.db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	s.db.SetMaxOpenConns(1)
	s.db.SetMaxIdleConns(1)

	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 500;`,
		`PRAGMA synchronous = NORMAL;`,
	}

	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			_ = s.db.Close()
			s.db = nil
			return err
		}
	}
	if err := s.db.Ping(); err != nil {
		_ = s.db.Close()
		s.db = nil
		return err
	}

	logger.Printf("successfully connected to %s", dbPath)

	return nil
}

func InitDatabase(dbPath string, overwrite bool) error {
	if overwrite {
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS audit_jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,

    request_id TEXT NOT NULL,
    host TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    query TEXT,
    upstream TEXT NOT NULL,
    status INTEGER,
    timestamp TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,

    headers TEXT,
    body TEXT,
    error TEXT
);
`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS findings (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,

    rule_id TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,

    created_at TEXT NOT NULL,

    FOREIGN KEY (job_id) REFERENCES audit_jobs(id)
);
`)
	if err != nil {
		return err
	}

	return nil
}

func newUUID() string {
	return uuid.NewString()
}

func marshalHeaders(h http.Header) (string, error) {
	jsonData, err := json.Marshal(h)
	return string(jsonData), err
}
