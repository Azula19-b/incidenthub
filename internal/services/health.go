package services

import "github.com/Azula19-b/incidenthub/internal/models"

func ImpactForSeverity(severity models.Severity) models.ServiceStatus {
	switch severity {
	case models.SeverityCritical:
		return models.StatusMajor
	case models.SeverityHigh:
		return models.StatusPartial
	case models.SeverityMedium:
		return models.StatusDegraded
	default:
		return models.StatusOperational
	}
}

func CalculateServiceStatus(activeIncidents []models.Incident) models.ServiceStatus {
	mostSevereRank := 0
	for _, incident := range activeIncidents {
		if incident.Status == models.IncidentResolved {
			continue
		}
		rank := SeverityRank(incident.Severity)
		if rank > mostSevereRank {
			mostSevereRank = rank
		}
	}
	switch mostSevereRank {
	case 4:
		return models.StatusMajor
	case 3:
		return models.StatusPartial
	case 2:
		return models.StatusDegraded
	default:
		return models.StatusOperational
	}
}

func SeverityRank(severity models.Severity) int {
	switch severity {
	case models.SeverityCritical:
		return 4
	case models.SeverityHigh:
		return 3
	case models.SeverityMedium:
		return 2
	case models.SeverityLow:
		return 1
	default:
		return 0
	}
}
