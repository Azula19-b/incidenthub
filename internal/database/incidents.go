package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azula19-b/incidenthub/internal/models"
)

type IncidentFilter struct {
	Search    string
	Severity  string
	Status    string
	ServiceID int64
	Sort      string
}

func (s *Store) ListIncidents(ctx context.Context, filter IncidentFilter) ([]models.Incident, error) {
	query := `SELECT DISTINCT i.id, i.title, i.description, i.severity, i.status,
        i.reporter, i.assigned_owner, i.created_at, i.updated_at, i.resolved_at
        FROM incidents i`
	var conditions []string
	var args []any

	if filter.ServiceID > 0 {
		query += ` JOIN incident_services filter_is ON filter_is.incident_id = i.id`
		conditions = append(conditions, `filter_is.service_id = ?`)
		args = append(args, filter.ServiceID)
	}
	if filter.Severity != "" {
		conditions = append(conditions, `i.severity = ?`)
		args = append(args, filter.Severity)
	}
	if filter.Status != "" {
		conditions = append(conditions, `i.status = ?`)
		args = append(args, filter.Status)
	}
	if strings.TrimSpace(filter.Search) != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		conditions = append(conditions, `(LOWER(i.title) LIKE ? OR LOWER(i.description) LIKE ?
            OR LOWER(i.reporter) LIKE ? OR LOWER(i.assigned_owner) LIKE ? OR EXISTS (
                SELECT 1 FROM incident_services search_is JOIN services search_s ON search_s.id = search_is.service_id
                WHERE search_is.incident_id = i.id AND LOWER(search_s.name) LIKE ?
            ))`)
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	query += incidentOrderClause(filter.Sort)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []models.Incident
	for rows.Next() {
		incident, scanErr := scanIncident(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		incidents = append(incidents, incident)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range incidents {
		services, loadErr := s.incidentServices(ctx, incidents[index].ID)
		if loadErr != nil {
			return nil, loadErr
		}
		incidents[index].Services = services
	}
	return incidents, nil
}

func (s *Store) GetIncident(ctx context.Context, id int64) (models.Incident, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, description, severity, status,
        reporter, assigned_owner, created_at, updated_at, resolved_at FROM incidents WHERE id = ?`, id)
	incident, err := scanIncident(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Incident{}, ErrNotFound
	}
	if err != nil {
		return models.Incident{}, fmt.Errorf("get incident: %w", err)
	}
	incident.Services, err = s.incidentServices(ctx, id)
	return incident, err
}

func (s *Store) CreateIncident(ctx context.Context, incident *models.Incident, serviceIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Truncate(time.Second)
	var resolved any
	if incident.Status == models.IncidentResolved {
		resolved = formatTime(now)
		incident.ResolvedAt = &now
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO incidents
        (title, description, severity, status, reporter, assigned_owner, created_at, updated_at, resolved_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, incident.Title, incident.Description,
		incident.Severity, incident.Status, incident.Reporter, incident.AssignedOwner,
		formatTime(now), formatTime(now), resolved)
	if err != nil {
		return fmt.Errorf("create incident: %w", err)
	}
	incident.ID, _ = result.LastInsertId()
	if err := linkIncidentServices(ctx, tx, incident.ID, serviceIDs); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO timeline_updates
        (incident_id, message, status, created_at) VALUES (?, ?, ?, ?)`, incident.ID,
		"Incident reported and response team engaged.", incident.Status, formatTime(now)); err != nil {
		return fmt.Errorf("create initial timeline update: %w", err)
	}
	incident.CreatedAt, incident.UpdatedAt = now, now
	return tx.Commit()
}

func (s *Store) UpdateIncident(ctx context.Context, incident models.Incident, serviceIDs []int64) error {
	existing, err := s.GetIncident(ctx, incident.ID)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Truncate(time.Second)
	var resolved any
	if incident.Status == models.IncidentResolved {
		resolvedTime := now
		if existing.ResolvedAt != nil {
			resolvedTime = *existing.ResolvedAt
		}
		resolved = formatTime(resolvedTime)
	}
	result, err := tx.ExecContext(ctx, `UPDATE incidents SET title = ?, description = ?, severity = ?,
        status = ?, reporter = ?, assigned_owner = ?, updated_at = ?, resolved_at = ? WHERE id = ?`,
		incident.Title, incident.Description, incident.Severity, incident.Status, incident.Reporter,
		incident.AssignedOwner, formatTime(now), resolved, incident.ID)
	if err != nil {
		return fmt.Errorf("update incident: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM incident_services WHERE incident_id = ?`, incident.ID); err != nil {
		return fmt.Errorf("replace incident services: %w", err)
	}
	if err := linkIncidentServices(ctx, tx, incident.ID, serviceIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteIncident(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM incidents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete incident: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) incidentServices(ctx context.Context, incidentID int64) ([]models.Service, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.name, s.description, s.owner_team, s.status,
        s.created_at, s.updated_at FROM services s JOIN incident_services isv ON isv.service_id = s.id
        WHERE isv.incident_id = ? ORDER BY s.name`, incidentID)
	if err != nil {
		return nil, fmt.Errorf("load incident services: %w", err)
	}
	defer rows.Close()
	var services []models.Service
	for rows.Next() {
		service, scanErr := scanService(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		services = append(services, service)
	}
	return services, rows.Err()
}

func linkIncidentServices(ctx context.Context, tx *sql.Tx, incidentID int64, serviceIDs []int64) error {
	for _, serviceID := range serviceIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO incident_services (incident_id, service_id) VALUES (?, ?)`,
			incidentID, serviceID); err != nil {
			if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
				return ErrNotFound
			}
			return fmt.Errorf("link incident service: %w", err)
		}
	}
	return nil
}

func scanIncident(row rowScanner) (models.Incident, error) {
	var incident models.Incident
	var created, updated string
	var resolved sql.NullString
	if err := row.Scan(&incident.ID, &incident.Title, &incident.Description, &incident.Severity,
		&incident.Status, &incident.Reporter, &incident.AssignedOwner, &created, &updated, &resolved); err != nil {
		return models.Incident{}, err
	}
	incident.CreatedAt, _ = time.Parse(time.RFC3339, created)
	incident.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	if resolved.Valid {
		value, _ := time.Parse(time.RFC3339, resolved.String)
		incident.ResolvedAt = &value
	}
	return incident, nil
}

func incidentOrderClause(sort string) string {
	switch sort {
	case "oldest":
		return ` ORDER BY i.created_at ASC`
	case "severity-asc":
		return ` ORDER BY CASE i.severity WHEN 'SEV-4' THEN 1 WHEN 'SEV-3' THEN 2 WHEN 'SEV-2' THEN 3 ELSE 4 END ASC, i.created_at DESC`
	case "severity-desc":
		return ` ORDER BY CASE i.severity WHEN 'SEV-1' THEN 1 WHEN 'SEV-2' THEN 2 WHEN 'SEV-3' THEN 3 ELSE 4 END ASC, i.created_at DESC`
	default:
		return ` ORDER BY i.created_at DESC`
	}
}
