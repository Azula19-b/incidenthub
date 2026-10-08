package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Azula19-b/incidenthub/internal/models"
)

func seededStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "incidenthub-test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	if err := store.Seed(context.Background()); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}
	if err := store.RecalculateAllServiceStatuses(context.Background()); err != nil {
		t.Fatalf("health calculation error: %v", err)
	}
	return store
}

func TestIncidentFilteringAndCaseInsensitiveSearch(t *testing.T) {
	store := seededStore(t)
	ctx := context.Background()
	incidents, err := store.ListIncidents(ctx, IncidentFilter{Search: "PAYMENT", Severity: "SEV-2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(incidents) != 1 || incidents[0].Title != "Elevated payment authorization failures" {
		t.Fatalf("unexpected filtered incidents: %#v", incidents)
	}
	byService, err := store.ListIncidents(ctx, IncidentFilter{ServiceID: incidents[0].Services[0].ID})
	if err != nil || len(byService) == 0 {
		t.Fatalf("service filter returned %d incidents, error %v", len(byService), err)
	}
}

func TestIncidentSorting(t *testing.T) {
	store := seededStore(t)
	ctx := context.Background()
	newest, err := store.ListIncidents(ctx, IncidentFilter{Sort: "newest"})
	if err != nil {
		t.Fatal(err)
	}
	oldest, err := store.ListIncidents(ctx, IncidentFilter{Sort: "oldest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(newest) < 2 || !newest[0].CreatedAt.After(newest[len(newest)-1].CreatedAt) {
		t.Fatal("newest sort order is incorrect")
	}
	if !oldest[0].CreatedAt.Before(oldest[len(oldest)-1].CreatedAt) {
		t.Fatal("oldest sort order is incorrect")
	}
	highest, err := store.ListIncidents(ctx, IncidentFilter{Sort: "severity-desc"})
	if err != nil {
		t.Fatal(err)
	}
	if highest[0].Severity != models.SeverityCritical {
		t.Fatalf("highest severity sort begins with %s", highest[0].Severity)
	}
}

func TestAnalyticsAndAutomaticServiceHealth(t *testing.T) {
	store := seededStore(t)
	ctx := context.Background()
	analytics, err := store.Analytics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if analytics.TotalIncidents != 5 || analytics.ActiveIncidents != 2 || analytics.ResolvedIncidents != 3 {
		t.Fatalf("unexpected analytics totals: %+v", analytics)
	}
	services, err := store.ListServices(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	statusByName := make(map[string]models.ServiceStatus)
	for _, service := range services {
		statusByName[service.Name] = service.Status
	}
	if statusByName["Payment Service"] != models.StatusPartial {
		t.Fatalf("Payment Service status = %q, want Partial Outage", statusByName["Payment Service"])
	}
	if statusByName["Notification Service"] != models.StatusDegraded {
		t.Fatalf("Notification Service status = %q, want Degraded Performance", statusByName["Notification Service"])
	}
}
