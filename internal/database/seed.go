package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type seedIncident struct {
	title, description, severity, status, reporter, owner string
	serviceIndexes                                        []int
	ageHours, resolvedAfterHours                          int
}

func (s *Store) Seed(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM services`).Scan(&count); err != nil {
		return fmt.Errorf("check seed state: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	services := [][3]string{
		{"API Gateway", "Routes and authenticates all public API traffic.", "Platform Engineering"},
		{"Authentication Service", "Identity, sessions, and access token management.", "Identity Team"},
		{"Payment Service", "Payment authorization, capture, and reconciliation.", "Commerce Engineering"},
		{"Notification Service", "Transactional email, SMS, and webhook delivery.", "Engagement Platform"},
		{"Primary Database", "Primary transactional PostgreSQL cluster.", "Data Infrastructure"},
		{"File Storage Service", "Secure customer document and media storage.", "Storage Platform"},
	}

	serviceIDs := make([]int64, 0, len(services))
	now := time.Now().UTC().Truncate(time.Second)
	for _, service := range services {
		result, execErr := tx.ExecContext(ctx, `
            INSERT INTO services (name, description, owner_team, status, created_at, updated_at)
            VALUES (?, ?, ?, 'Operational', ?, ?)`,
			service[0], service[1], service[2], formatTime(now.AddDate(0, -5, 0)), formatTime(now))
		if execErr != nil {
			return fmt.Errorf("seed service %q: %w", service[0], execErr)
		}
		id, _ := result.LastInsertId()
		serviceIDs = append(serviceIDs, id)
	}

	incidents := []seedIncident{
		{"Elevated payment authorization failures", "Payment authorizations are timing out for a subset of card networks.", "SEV-2", "Identified", "Nora Chen", "Commerce On-Call", []int{2, 0}, 3, 0},
		{"Delayed notification delivery", "Outbound email and webhook queues are experiencing increased latency.", "SEV-3", "Monitoring", "Marcus Webb", "Engagement On-Call", []int{3}, 8, 0},
		{"Database connection saturation", "A deployment exhausted the application connection pool and increased API error rates.", "SEV-1", "Resolved", "Priya Shah", "Data Infrastructure", []int{4, 0}, 72, 2},
		{"Intermittent sign-in failures", "A cache node caused intermittent session validation failures in one region.", "SEV-2", "Resolved", "Elena Garcia", "Identity On-Call", []int{1}, 168, 1},
		{"File preview generation backlog", "Preview workers accumulated a backlog after a traffic spike.", "SEV-4", "Resolved", "Sam Okafor", "Storage Platform", []int{5}, 360, 5},
	}

	for _, incident := range incidents {
		if err := seedOneIncident(ctx, tx, incident, serviceIDs, now); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed data: %w", err)
	}
	return nil
}

func seedOneIncident(ctx context.Context, tx *sql.Tx, incident seedIncident, serviceIDs []int64, now time.Time) error {
	created := now.Add(-time.Duration(incident.ageHours) * time.Hour)
	updated := created.Add(25 * time.Minute)
	var resolved any
	if incident.status == "Resolved" {
		resolvedTime := created.Add(time.Duration(incident.resolvedAfterHours) * time.Hour)
		resolved = formatTime(resolvedTime)
		updated = resolvedTime
	}

	result, err := tx.ExecContext(ctx, `
        INSERT INTO incidents
            (title, description, severity, status, reporter, assigned_owner, created_at, updated_at, resolved_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, incident.title, incident.description, incident.severity,
		incident.status, incident.reporter, incident.owner, formatTime(created), formatTime(updated), resolved)
	if err != nil {
		return fmt.Errorf("seed incident %q: %w", incident.title, err)
	}
	incidentID, _ := result.LastInsertId()

	for _, index := range incident.serviceIndexes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO incident_services (incident_id, service_id) VALUES (?, ?)`,
			incidentID, serviceIDs[index]); err != nil {
			return fmt.Errorf("link seeded incident to service: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO timeline_updates (incident_id, message, status, created_at)
        VALUES (?, ?, 'Investigating', ?)`, incidentID, "Incident reported and response team engaged.", formatTime(created)); err != nil {
		return fmt.Errorf("seed incident timeline: %w", err)
	}
	if incident.status != "Investigating" {
		message := "Investigation progressed and incident status updated."
		if incident.status == "Resolved" {
			message = "Mitigation completed and service health restored."
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO timeline_updates (incident_id, message, status, created_at)
            VALUES (?, ?, ?, ?)`, incidentID, message, incident.status, formatTime(updated)); err != nil {
			return fmt.Errorf("seed incident resolution timeline: %w", err)
		}
	}
	return nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
