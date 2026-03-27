// Package storage - repository pattern for profile CRUD operations on SQLite.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ProfileRow maps to the profiles SQLite table.
type ProfileRow struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Platform   string  `json:"platform"`
	Channel    string  `json:"channel"`
	Active     bool    `json:"active"`
	MaxWorkers *int    `json:"maxWorkers,omitempty"`
	Features   *string `json:"features,omitempty"` // JSON string
	TokenIDs   *string `json:"tokenIds,omitempty"` // JSON array string
	ProxyTag   *string `json:"proxyTag,omitempty"`
	Notes      *string `json:"notes,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

// ProfileRepo handles profile CRUD in SQLite.
type ProfileRepo struct {
	db *sql.DB
}

// NewProfileRepo creates a new profile repository.
func NewProfileRepo(db *DB) *ProfileRepo {
	return &ProfileRepo{db: db.Conn()}
}

// Create inserts a new profile.
func (r *ProfileRepo) Create(ctx context.Context, p *ProfileRow) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO profiles (id, name, platform, channel, active, max_workers, features, token_ids, proxy_tag, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Platform, p.Channel, p.Active, p.MaxWorkers, p.Features, p.TokenIDs, p.ProxyTag, p.Notes,
	)
	return err
}

// GetByID retrieves a profile by its ID.
func (r *ProfileRepo) GetByID(ctx context.Context, id string) (*ProfileRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, platform, channel, active, max_workers, features, token_ids, proxy_tag, notes, created_at, updated_at
		 FROM profiles WHERE id = ?`, id,
	)
	return r.scanRow(row)
}

// GetActive returns the currently active profile.
func (r *ProfileRepo) GetActive(ctx context.Context) (*ProfileRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, platform, channel, active, max_workers, features, token_ids, proxy_tag, notes, created_at, updated_at
		 FROM profiles WHERE active = 1 LIMIT 1`,
	)
	p, err := r.scanRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// List returns all profiles ordered by creation date.
func (r *ProfileRepo) List(ctx context.Context) ([]ProfileRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, platform, channel, active, max_workers, features, token_ids, proxy_tag, notes, created_at, updated_at
		 FROM profiles ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []ProfileRow
	for rows.Next() {
		p, err := r.scanRows(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *p)
	}
	return profiles, rows.Err()
}

// Update modifies an existing profile.
func (r *ProfileRepo) Update(ctx context.Context, p *ProfileRow) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE profiles SET name=?, platform=?, channel=?, active=?, max_workers=?, features=?, token_ids=?, proxy_tag=?, notes=?, updated_at=?
		 WHERE id=?`,
		p.Name, p.Platform, p.Channel, p.Active, p.MaxWorkers, p.Features, p.TokenIDs, p.ProxyTag, p.Notes, time.Now().Format(time.RFC3339), p.ID,
	)
	return err
}

// Delete removes a profile by ID.
func (r *ProfileRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("profile %q not found", id)
	}
	return nil
}

// SetActive deactivates all profiles then activates the specified one.
func (r *ProfileRepo) SetActive(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET active = 0`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET active = 1 WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ExistsByChannel checks if a profile already exists for the given platform+channel.
func (r *ProfileRepo) ExistsByChannel(ctx context.Context, platform, channel string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM profiles WHERE platform = ? AND channel = ?`,
		platform, channel,
	).Scan(&count)
	return count > 0, err
}

// CountByPlatform returns the number of profiles per platform.
func (r *ProfileRepo) CountByPlatform(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT platform, COUNT(*) FROM profiles GROUP BY platform`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var platform string
		var count int
		if err := rows.Scan(&platform, &count); err != nil {
			return nil, err
		}
		counts[platform] = count
	}
	return counts, rows.Err()
}

func (r *ProfileRepo) scanRow(row *sql.Row) (*ProfileRow, error) {
	p := &ProfileRow{}
	err := row.Scan(&p.ID, &p.Name, &p.Platform, &p.Channel, &p.Active,
		&p.MaxWorkers, &p.Features, &p.TokenIDs, &p.ProxyTag, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProfileRepo) scanRows(rows *sql.Rows) (*ProfileRow, error) {
	p := &ProfileRow{}
	err := rows.Scan(&p.ID, &p.Name, &p.Platform, &p.Channel, &p.Active,
		&p.MaxWorkers, &p.Features, &p.TokenIDs, &p.ProxyTag, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetMetricsTimeline returns metrics snapshots for a session, useful for charts.
func (db *DB) GetMetricsTimeline(ctx context.Context, sessionID int64) ([]MetricsSnapshot, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT timestamp, active_viewers, total_workers, segments, bytes_received, heartbeats, ads_watched
		 FROM metrics_snapshots WHERE session_id = ? ORDER BY timestamp ASC`, sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []MetricsSnapshot
	for rows.Next() {
		var s MetricsSnapshot
		if err := rows.Scan(&s.Timestamp, &s.ActiveViewers, &s.TotalWorkers,
			&s.Segments, &s.BytesReceived, &s.Heartbeats, &s.AdsWatched); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

// MetricsSnapshot represents a point-in-time metrics record.
type MetricsSnapshot struct {
	Timestamp     string `json:"timestamp"`
	ActiveViewers int    `json:"activeViewers"`
	TotalWorkers  int    `json:"totalWorkers"`
	Segments      int    `json:"segments"`
	BytesReceived int64  `json:"bytesReceived"`
	Heartbeats    int    `json:"heartbeats"`
	AdsWatched    int    `json:"adsWatched"`
}

// GetSessionStats returns aggregate statistics across all sessions.
func (db *DB) GetSessionStats(ctx context.Context) (*SessionStats, error) {
	var stats SessionStats
	err := db.conn.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_segments), 0), COALESCE(SUM(total_bytes), 0),
			COALESCE(SUM(total_ads), 0), COALESCE(MAX(max_viewers), 0)
		 FROM sessions`,
	).Scan(&stats.TotalSessions, &stats.TotalSegments, &stats.TotalBytes,
		&stats.TotalAds, &stats.PeakViewers)
	return &stats, err
}

// SessionStats holds aggregate session statistics.
type SessionStats struct {
	TotalSessions int   `json:"totalSessions"`
	TotalSegments int64 `json:"totalSegments"`
	TotalBytes    int64 `json:"totalBytes"`
	TotalAds      int   `json:"totalAds"`
	PeakViewers   int   `json:"peakViewers"`
}

// CleanupOldSessions removes sessions older than the given duration.
func (db *DB) CleanupOldSessions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)
	result, err := db.conn.ExecContext(ctx,
		`DELETE FROM sessions WHERE ended_at IS NOT NULL AND ended_at < ?`, cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Purge deletes JSON features field into structured data (utility for migration).
func ParseFeaturesJSON(raw *string) (map[string]bool, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	var features map[string]bool
	if err := json.Unmarshal([]byte(*raw), &features); err != nil {
		return nil, err
	}
	return features, nil
}
