package validation

import (
	"testing"

	"github.com/Azula19-b/incidenthub/internal/models"
)

func TestValidateIncidentAcceptsCompleteInput(t *testing.T) {
	errors := ValidateIncident(IncidentInput{
		Title: "Checkout failures", Description: "Customers cannot complete checkout.",
		Reporter: "Support", AssignedOwner: "Commerce On-Call", Severity: models.SeverityHigh,
		Status: models.IncidentInvestigating, ServiceIDs: []int64{1, 2},
	})
	if len(errors) != 0 {
		t.Fatalf("expected valid incident, got errors: %v", errors)
	}
}

func TestValidateIncidentRejectsMissingAndInvalidValues(t *testing.T) {
	errors := ValidateIncident(IncidentInput{Severity: "SEV-9", Status: "Closed"})
	for _, field := range []string{"title", "description", "reporter", "assigned_owner", "severity", "status", "service_ids"} {
		if errors[field] == "" {
			t.Errorf("expected validation error for %s", field)
		}
	}
}

func TestValidateServiceRejectsInvalidInput(t *testing.T) {
	errors := ValidateService(ServiceInput{Name: " ", Description: "", OwnerTeam: "", Status: "Unknown"})
	if len(errors) != 4 {
		t.Fatalf("expected 4 service validation errors, got %v", errors)
	}
}
