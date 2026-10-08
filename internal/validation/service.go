package validation

import (
	"strings"

	"github.com/Azula19-b/incidenthub/internal/models"
)

type ServiceInput struct {
	Name        string
	Description string
	OwnerTeam   string
	Status      models.ServiceStatus
}

func ValidateService(input ServiceInput) map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(input.Name) == "" {
		errors["name"] = "Service name is required."
	} else if len([]rune(input.Name)) > 100 {
		errors["name"] = "Service name must be 100 characters or fewer."
	}
	if strings.TrimSpace(input.Description) == "" {
		errors["description"] = "Description is required."
	} else if len([]rune(input.Description)) > 1000 {
		errors["description"] = "Description must be 1000 characters or fewer."
	}
	if strings.TrimSpace(input.OwnerTeam) == "" {
		errors["owner_team"] = "Owner or team is required."
	} else if len([]rune(input.OwnerTeam)) > 100 {
		errors["owner_team"] = "Owner or team must be 100 characters or fewer."
	}
	if !validServiceStatus(input.Status) {
		errors["status"] = "Select a valid service status."
	}
	return errors
}

func validServiceStatus(status models.ServiceStatus) bool {
	switch status {
	case models.StatusOperational, models.StatusDegraded, models.StatusPartial,
		models.StatusMajor, models.StatusMaintenance:
		return true
	default:
		return false
	}
}
