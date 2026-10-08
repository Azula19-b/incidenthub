package database

import (
	"context"
	"fmt"
	"math"

	"github.com/Azula19-b/incidenthub/internal/models"
)

func (s *Store) DashboardMetrics(ctx context.Context) (models.DashboardMetrics, error) {
	var metrics models.DashboardMetrics
	err := s.db.QueryRowContext(ctx, `SELECT
        SUM(CASE WHEN status <> 'Resolved' THEN 1 ELSE 0 END),
        SUM(CASE WHEN status <> 'Resolved' AND severity = 'SEV-1' THEN 1 ELSE 0 END),
        SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END),
        COALESCE(AVG(CASE WHEN resolved_at IS NOT NULL THEN
          (julianday(resolved_at) - julianday(created_at)) * 24 END), 0)
        FROM incidents`).Scan(&metrics.ActiveIncidents, &metrics.CriticalIncidents,
		&metrics.ResolvedIncidents, &metrics.AverageResolution)
	if err != nil {
		return metrics, fmt.Errorf("load dashboard incident metrics: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM services`).Scan(&metrics.MonitoredServices); err != nil {
		return metrics, fmt.Errorf("count monitored services: %w", err)
	}
	metrics.OverallStatus = models.StatusOperational
	rows, err := s.db.QueryContext(ctx, `SELECT status FROM services`)
	if err != nil {
		return metrics, err
	}
	defer rows.Close()
	currentRank := 0
	for rows.Next() {
		var status models.ServiceStatus
		if err := rows.Scan(&status); err != nil {
			return metrics, err
		}
		if rank := serviceStatusRank(status); rank > currentRank {
			currentRank, metrics.OverallStatus = rank, status
		}
	}
	return metrics, rows.Err()
}

func (s *Store) Analytics(ctx context.Context) (models.Analytics, error) {
	var analytics models.Analytics
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),
        SUM(CASE WHEN status <> 'Resolved' THEN 1 ELSE 0 END),
        SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END),
        COALESCE(AVG(CASE WHEN resolved_at IS NOT NULL THEN
          (julianday(resolved_at) - julianday(created_at)) * 24 END), 0),
        SUM(CASE WHEN datetime(created_at) >= datetime('now', '-7 days') THEN 1 ELSE 0 END),
        SUM(CASE WHEN datetime(created_at) >= datetime('now', '-30 days') THEN 1 ELSE 0 END)
        FROM incidents`).Scan(&analytics.TotalIncidents, &analytics.ActiveIncidents,
		&analytics.ResolvedIncidents, &analytics.AverageResolutionHrs,
		&analytics.CreatedLast7Days, &analytics.CreatedLast30Days)
	if err != nil {
		return analytics, fmt.Errorf("load incident analytics: %w", err)
	}

	severityCounts := map[models.Severity]int{}
	rows, err := s.db.QueryContext(ctx, `SELECT severity, COUNT(*) FROM incidents GROUP BY severity`)
	if err != nil {
		return analytics, err
	}
	for rows.Next() {
		var severity models.Severity
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			rows.Close()
			return analytics, err
		}
		severityCounts[severity] = count
	}
	rows.Close()
	for _, severity := range []models.Severity{models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow} {
		count := severityCounts[severity]
		analytics.SeverityDistribution = append(analytics.SeverityDistribution, models.SeverityMetric{
			Severity: severity, Count: count, Percentage: percentage(count, analytics.TotalIncidents),
		})
	}

	rows, err = s.db.QueryContext(ctx, `SELECT s.id, s.name, COUNT(isv.incident_id) AS incident_count
        FROM services s LEFT JOIN incident_services isv ON isv.service_id = s.id
        GROUP BY s.id, s.name ORDER BY incident_count DESC, s.name`)
	if err != nil {
		return analytics, err
	}
	maxCount := 0
	for rows.Next() {
		var metric models.ServiceIncidentMetric
		if err := rows.Scan(&metric.ServiceID, &metric.ServiceName, &metric.Count); err != nil {
			rows.Close()
			return analytics, err
		}
		if metric.Count > maxCount {
			maxCount = metric.Count
		}
		analytics.IncidentsByService = append(analytics.IncidentsByService, metric)
	}
	rows.Close()
	for index := range analytics.IncidentsByService {
		analytics.IncidentsByService[index].Percentage = percentage(analytics.IncidentsByService[index].Count, maxCount)
	}
	if len(analytics.IncidentsByService) > 0 && analytics.IncidentsByService[0].Count > 0 {
		analytics.MostAffectedService = analytics.IncidentsByService[0].ServiceName
	} else {
		analytics.MostAffectedService = "No incident history"
	}
	analytics.AverageResolutionHrs = math.Round(analytics.AverageResolutionHrs*10) / 10
	return analytics, nil
}

func percentage(value, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round((float64(value)/float64(total))*1000) / 10
}

func serviceStatusRank(status models.ServiceStatus) int {
	switch status {
	case models.StatusMajor:
		return 4
	case models.StatusPartial:
		return 3
	case models.StatusDegraded:
		return 2
	case models.StatusMaintenance:
		return 1
	default:
		return 0
	}
}
