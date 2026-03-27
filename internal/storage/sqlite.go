// Package storage provides SQLite-backed persistence for profiles, metrics history, and sessions.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection with migration support.
type DB struct {
	conn   *sql.DB
	logger *slog.Logger
}

// Open creates or opens a SQLite database at the given path.
func Open(dbPath string, logger *slog.Logger) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode and foreign keys
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	db := &DB{
		conn:   conn,
		logger: logger.With("component", "storage"),
	}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying sql.DB for custom queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// migrate runs the schema migrations.
func (db *DB) migrate() error {
	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}
	db.logger.Info("database migrations applied")
	return nil
}

// schema is the full database schema (idempotent with IF NOT EXISTS).
const schema = `
CREATE TABLE IF NOT EXISTS profiles (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    platform    TEXT NOT NULL DEFAULT 'twitch',
    channel     TEXT NOT NULL,
    active      INTEGER NOT NULL DEFAULT 0,
    max_workers INTEGER,
    features    TEXT,
    token_ids   TEXT,
    proxy_tag   TEXT,
    notes       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id    TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    channel       TEXT NOT NULL,
    platform      TEXT NOT NULL,
    started_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at      DATETIME,
    max_viewers   INTEGER DEFAULT 0,
    total_segments INTEGER DEFAULT 0,
    total_bytes   INTEGER DEFAULT 0,
    total_ads     INTEGER DEFAULT 0,
    total_heartbeats INTEGER DEFAULT 0,
    end_reason    TEXT
);

CREATE TABLE IF NOT EXISTS metrics_snapshots (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    timestamp       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    active_viewers  INTEGER DEFAULT 0,
    total_workers   INTEGER DEFAULT 0,
    segments        INTEGER DEFAULT 0,
    bytes_received  INTEGER DEFAULT 0,
    heartbeats      INTEGER DEFAULT 0,
    ads_watched     INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS proxy_health_log (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    host      TEXT NOT NULL,
    port      TEXT NOT NULL,
    health    INTEGER NOT NULL,
    latency   INTEGER,
    ip        TEXT,
    checked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_profile ON sessions(profile_id);
CREATE INDEX IF NOT EXISTS idx_metrics_session ON metrics_snapshots(session_id);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics_snapshots(timestamp);
CREATE INDEX IF NOT EXISTS idx_proxy_health_checked ON proxy_health_log(checked_at);
`

// InsertSession creates a new session record and returns its ID.
func (db *DB) InsertSession(ctx context.Context, profileID, channel, platform string) (int64, error) {
	result, err := db.conn.ExecContext(ctx,
		"INSERT INTO sessions (profile_id, channel, platform) VALUES (?, ?, ?)",
		profileID, channel, platform,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// EndSession marks a session as finished with final metrics.
func (db *DB) EndSession(ctx context.Context, sessionID int64, reason string, maxViewers, totalSegments int, totalBytes int64, totalAds, totalHeartbeats int) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE sessions SET
			ended_at = CURRENT_TIMESTAMP,
			end_reason = ?,
			max_viewers = ?,
			total_segments = ?,
			total_bytes = ?,
			total_ads = ?,
			total_heartbeats = ?
		WHERE id = ?`,
		reason, maxViewers, totalSegments, totalBytes, totalAds, totalHeartbeats, sessionID,
	)
	return err
}

// InsertMetricsSnapshot records a point-in-time metrics snapshot.
func (db *DB) InsertMetricsSnapshot(ctx context.Context, sessionID int64, activeViewers, totalWorkers, segments int, bytesReceived int64, heartbeats, ads int) error {
	_, err := db.conn.ExecContext(ctx,
		`INSERT INTO metrics_snapshots (session_id, active_viewers, total_workers, segments, bytes_received, heartbeats, ads_watched)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionID, activeViewers, totalWorkers, segments, bytesReceived, heartbeats, ads,
	)
	return err
}

// GetRecentSessions returns the N most recent sessions.
func (db *DB) GetRecentSessions(ctx context.Context, limit int) ([]Session, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, profile_id, channel, platform, started_at, ended_at,
			max_viewers, total_segments, total_bytes, total_ads, total_heartbeats, end_reason
		FROM sessions ORDER BY started_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(&s.ID, &s.ProfileID, &s.Channel, &s.Platform,
			&s.StartedAt, &s.EndedAt, &s.MaxViewers, &s.TotalSegments,
			&s.TotalBytes, &s.TotalAds, &s.TotalHeartbeats, &s.EndReason,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Session represents a stored session record.
type Session struct {
	ID              int64   `json:"id"`
	ProfileID       string  `json:"profileId"`
	Channel         string  `json:"channel"`
	Platform        string  `json:"platform"`
	StartedAt       string  `json:"startedAt"`
	EndedAt         *string `json:"endedAt"`
	MaxViewers      int     `json:"maxViewers"`
	TotalSegments   int     `json:"totalSegments"`
	TotalBytes      int64   `json:"totalBytes"`
	TotalAds        int     `json:"totalAds"`
	TotalHeartbeats int     `json:"totalHeartbeats"`
	EndReason       *string `json:"endReason"`
}
