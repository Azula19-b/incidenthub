package models

import "time"

type ServiceStatus string

const (
	StatusOperational ServiceStatus = "Operational"
	StatusDegraded    ServiceStatus = "Degraded Performance"
	StatusPartial     ServiceStatus = "Partial Outage"
	StatusMajor       ServiceStatus = "Major Outage"
	StatusMaintenance ServiceStatus = "Maintenance"
)

type Severity string

const (
	SeverityCritical Severity = "SEV-1"
	SeverityHigh     Severity = "SEV-2"
	SeverityMedium   Severity = "SEV-3"
	SeverityLow      Severity = "SEV-4"
)

type IncidentStatus string

const (
	IncidentInvestigating IncidentStatus = "Investigating"
	IncidentIdentified    IncidentStatus = "Identified"
	IncidentMonitoring    IncidentStatus = "Monitoring"
	IncidentResolved      IncidentStatus = "Resolved"
)

type Service struct {
	ID          int64
	Name        string
	Description string
	OwnerTeam   string
	Status      ServiceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Incident struct {
	ID            int64
	Title         string
	Description   string
	Severity      Severity
	Status        IncidentStatus
	Reporter      string
	AssignedOwner string
	Services      []Service
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ResolvedAt    *time.Time
}

type TimelineUpdate struct {
	ID         int64
	IncidentID int64
	Message    string
	Status     IncidentStatus
	CreatedAt  time.Time
}
