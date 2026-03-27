// Package storage - data export for session history (CSV and JSON formats).
package storage

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
)

// ExportFormat specifies the output format for data export.
type ExportFormat string

const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportSessions exports session history in the specified format.
func (db *DB) ExportSessions(ctx context.Context, format ExportFormat, limit int) ([]byte, error) {
	sessions, err := db.GetRecentSessions(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("fetching sessions: %w", err)
	}

	switch format {
	case FormatCSV:
		return exportSessionsCSV(sessions)
	case FormatJSON:
		return exportSessionsJSON(sessions)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportMetrics exports metrics timeline for a session.
func (db *DB) ExportMetrics(ctx context.Context, sessionID int64, format ExportFormat) ([]byte, error) {
	snapshots, err := db.GetMetricsTimeline(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetching metrics: %w", err)
	}

	switch format {
	case FormatCSV:
		return exportMetricsCSV(snapshots)
	case FormatJSON:
		return json.MarshalIndent(snapshots, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func exportSessionsCSV(sessions []Session) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header
	w.Write([]string{
		"ID", "ProfileID", "Channel", "Platform", "StartedAt", "EndedAt",
		"MaxViewers", "TotalSegments", "TotalBytes", "TotalAds", "TotalHeartbeats", "EndReason",
	})

	for _, s := range sessions {
		endedAt := ""
		if s.EndedAt != nil {
			endedAt = *s.EndedAt
		}
		endReason := ""
		if s.EndReason != nil {
			endReason = *s.EndReason
		}

		w.Write([]string{
			strconv.FormatInt(s.ID, 10),
			s.ProfileID,
			s.Channel,
			s.Platform,
			s.StartedAt,
			endedAt,
			strconv.Itoa(s.MaxViewers),
			strconv.Itoa(s.TotalSegments),
			strconv.FormatInt(s.TotalBytes, 10),
			strconv.Itoa(s.TotalAds),
			strconv.Itoa(s.TotalHeartbeats),
			endReason,
		})
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

func exportSessionsJSON(sessions []Session) ([]byte, error) {
	return json.MarshalIndent(sessions, "", "  ")
}

func exportMetricsCSV(snapshots []MetricsSnapshot) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	w.Write([]string{
		"Timestamp", "ActiveViewers", "TotalWorkers", "Segments",
		"BytesReceived", "Heartbeats", "AdsWatched",
	})

	for _, s := range snapshots {
		w.Write([]string{
			s.Timestamp,
			strconv.Itoa(s.ActiveViewers),
			strconv.Itoa(s.TotalWorkers),
			strconv.Itoa(s.Segments),
			strconv.FormatInt(s.BytesReceived, 10),
			strconv.Itoa(s.Heartbeats),
			strconv.Itoa(s.AdsWatched),
		})
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
