package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Azula19-b/incidenthub/internal/models"
)

func (s *Store) ListTimeline(ctx context.Context, incidentID int64) ([]models.TimelineUpdate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, incident_id, message, status, created_at
        FROM timeline_updates WHERE incident_id = ? ORDER BY created_at DESC, id DESC`, incidentID)
	if err != nil {
		return nil, fmt.Errorf("list timeline updates: %w", err)
	}
	defer rows.Close()
	var updates []models.TimelineUpdate
	for rows.Next() {
		var update models.TimelineUpdate
		var created string
		if err := rows.Scan(&update.ID, &update.IncidentID, &update.Message, &update.Status, &created); err != nil {
			return nil, err
		}
		update.CreatedAt, _ = time.Parse(time.RFC3339, created)
		updates = append(updates, update)
	}
	return updates, rows.Err()
}

func (s *Store) AddTimelineUpdate(ctx context.Context, incidentID int64, message string, status models.IncidentStatus) error {
	var existingStatus models.IncidentStatus
	var existingResolved sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT status, resolved_at FROM incidents WHERE id = ?`, incidentID).
		Scan(&existingStatus, &existingResolved); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := tx.ExecContext(ctx, `INSERT INTO timeline_updates
        (incident_id, message, status, created_at) VALUES (?, ?, ?, ?)`,
		incidentID, message, status, formatTime(now)); err != nil {
		return fmt.Errorf("add timeline update: %w", err)
	}
	var resolved any
	if status == models.IncidentResolved {
		if existingStatus == models.IncidentResolved && existingResolved.Valid {
			resolved = existingResolved.String
		} else {
			resolved = formatTime(now)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE incidents SET status = ?, updated_at = ?, resolved_at = ? WHERE id = ?`,
		status, formatTime(now), resolved, incidentID); err != nil {
		return fmt.Errorf("update incident from timeline: %w", err)
	}
	return tx.Commit()
}
