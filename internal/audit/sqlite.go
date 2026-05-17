package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

type Finding struct {
	ID        string
	JobID     string
	RuleID    string
	Title     string
	Message   string
	CreatedAt time.Time
}

type Store interface {
	SaveAuditResult(ctx context.Context, job Job, jobID string, findings []Finding) error
}

func (s *SQLiteStore) SaveAuditResult(ctx context.Context, job Job, jobID string, findings []Finding) error {
	tx, err := s.db.BeginTx(ctx, nil)
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
		jobType string
		meta    Metadata
		headers string
		body    string
		errStr  string
	)

	switch j := job.(type) {
	case *RequestJob:
		jobType = string(j.JobType())
		meta = j.Meta

		headers, err = marshalHeaders(j.Headers)
		if err != nil {
			return err
		}

		body = string(j.Body)
	case *ResponseJob:
		jobType = string(j.JobType())
		meta = j.Meta

		headers, err = marshalHeaders(j.Headers)
		if err != nil {
			return err
		}

		body = string(j.Body)
	case *FailureJob:
		jobType = string(j.JobType())
		meta = j.Meta

		errStr = j.Error
	default:
		return fmt.Errorf("unknown job type: %T", job)
	}

	_, err = tx.ExecContext(
		ctx,
		query,
		jobID,
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

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO findings (
		id,
		job_id,
		rule_id,
		title,
		message,
		created_at
		) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, finding := range findings {
		if _, err := stmt.ExecContext(
			ctx,
			finding.ID,
			finding.JobID,
			finding.RuleID,
			finding.Title,
			finding.Message,
			finding.CreatedAt.Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) Connect(configDir string, logger *log.Logger) error {
	dbPath := filepath.Join(configDir, "observer.db")

	var err error
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
