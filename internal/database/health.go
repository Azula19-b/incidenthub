package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Azula19-b/incidenthub/internal/models"
	serviceengine "github.com/Azula19-b/incidenthub/internal/services"
)

func (s *Store) RecalculateAllServiceStatuses(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM services`)
	if err != nil {
		return fmt.Errorf("list services for health calculation: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	return s.RecalculateServiceStatuses(ctx, ids)
}

func (s *Store) RecalculateServiceStatuses(ctx context.Context, serviceIDs []int64) error {
	seen := make(map[int64]bool)
	for _, serviceID := range serviceIDs {
		if serviceID < 1 || seen[serviceID] {
			continue
		}
		seen[serviceID] = true
		rows, err := s.db.QueryContext(ctx, `SELECT i.severity, i.status FROM incidents i
            JOIN incident_services isv ON isv.incident_id = i.id
            WHERE isv.service_id = ? AND i.status <> 'Resolved'`, serviceID)
		if err != nil {
			return fmt.Errorf("load active service incidents: %w", err)
		}
		var active []models.Incident
		for rows.Next() {
			var incident models.Incident
			if err := rows.Scan(&incident.Severity, &incident.Status); err != nil {
				rows.Close()
				return err
			}
			active = append(active, incident)
		}
		rows.Close()
		status := serviceengine.CalculateServiceStatus(active)
		if _, err := s.db.ExecContext(ctx, `UPDATE services SET status = ?, updated_at = ? WHERE id = ?`,
			status, formatTime(time.Now().UTC().Truncate(time.Second)), serviceID); err != nil {
			return fmt.Errorf("update calculated service status: %w", err)
		}
	}
	return nil
}
