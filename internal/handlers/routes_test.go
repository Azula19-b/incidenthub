package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testApp(t *testing.T) *App {
	t.Helper()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDirectory) })
	app, err := New(filepath.Join(t.TempDir(), "routes.db"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { app.Close() })
	return app
}

func TestMajorPagesRenderSeedData(t *testing.T) {
	app := testApp(t)
	routes := app.Routes()
	tests := []struct{ path, contains string }{
		{"/", "Operations Dashboard"}, {"/incidents", "Elevated payment authorization failures"},
		{"/services", "Payment Service"}, {"/analytics", "Incident Analytics"},
		{"/status", "IncidentHub System Status"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s returned %d", test.path, recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), test.contains) {
				t.Fatalf("GET %s missing %q", test.path, test.contains)
			}
		})
	}
}

func TestInvalidServiceFormReturnsValidationMessage(t *testing.T) {
	app := testApp(t)
	request := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader("name=&description=&owner_team=&status=Unknown"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("invalid form returned %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Service name is required") {
		t.Fatal("validation message not rendered")
	}
}

func TestUnknownPageUsesCustom404(t *testing.T) {
	app := testApp(t)
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "That page is off the map") {
		t.Fatalf("unexpected 404 response: %d %s", recorder.Code, recorder.Body.String())
	}
}
