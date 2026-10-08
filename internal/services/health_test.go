package services

import (
	"testing"

	"github.com/Azula19-b/incidenthub/internal/models"
)

func TestImpactForSeverity(t *testing.T) {
	tests := []struct {
		severity models.Severity
		want     models.ServiceStatus
	}{
		{models.SeverityCritical, models.StatusMajor},
		{models.SeverityHigh, models.StatusPartial},
		{models.SeverityMedium, models.StatusDegraded},
		{models.SeverityLow, models.StatusOperational},
	}
	for _, test := range tests {
		t.Run(string(test.severity), func(t *testing.T) {
			if got := ImpactForSeverity(test.severity); got != test.want {
				t.Fatalf("ImpactForSeverity(%q) = %q, want %q", test.severity, got, test.want)
			}
		})
	}
}

func TestCalculateServiceStatusUsesMostSevereActiveIncident(t *testing.T) {
	incidents := []models.Incident{
		{Severity: models.SeverityMedium, Status: models.IncidentMonitoring},
		{Severity: models.SeverityCritical, Status: models.IncidentResolved},
		{Severity: models.SeverityHigh, Status: models.IncidentInvestigating},
	}
	if got := CalculateServiceStatus(incidents); got != models.StatusPartial {
		t.Fatalf("CalculateServiceStatus() = %q, want %q", got, models.StatusPartial)
	}
}

func TestCalculateServiceStatusReturnsOperationalWithoutActiveImpact(t *testing.T) {
	incidents := []models.Incident{{Severity: models.SeverityCritical, Status: models.IncidentResolved}}
	if got := CalculateServiceStatus(incidents); got != models.StatusOperational {
		t.Fatalf("CalculateServiceStatus() = %q, want Operational", got)
	}
}
