package validation

import (
	"strings"

	"github.com/Azula19-b/incidenthub/internal/models"
)

type IncidentInput struct {
	Title, Description, Reporter, AssignedOwner string
	Severity                                    models.Severity
	Status                                      models.IncidentStatus
	ServiceIDs                                  []int64
}

func ValidateIncident(input IncidentInput) map[string]string {
	errors := make(map[string]string)
	requiredText(errors, "title", input.Title, 160, "Title")
	requiredText(errors, "description", input.Description, 3000, "Description")
	requiredText(errors, "reporter", input.Reporter, 100, "Reporter")
	requiredText(errors, "assigned_owner", input.AssignedOwner, 100, "Assigned owner")
	if !ValidSeverity(input.Severity) {
		errors["severity"] = "Select a valid severity."
	}
	if !ValidIncidentStatus(input.Status) {
		errors["status"] = "Select a valid incident status."
	}
	if len(input.ServiceIDs) == 0 {
		errors["service_ids"] = "Select at least one affected service."
	}
	return errors
}

func ValidSeverity(value models.Severity) bool {
	switch value {
	case models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow:
		return true
	default:
		return false
	}
}

func ValidIncidentStatus(value models.IncidentStatus) bool {
	switch value {
	case models.IncidentInvestigating, models.IncidentIdentified, models.IncidentMonitoring, models.IncidentResolved:
		return true
	default:
		return false
	}
}

func requiredText(errors map[string]string, field, value string, max int, label string) {
	if strings.TrimSpace(value) == "" {
		errors[field] = label + " is required."
	} else if len([]rune(value)) > max {
		errors[field] = label + " is too long."
	}
}
