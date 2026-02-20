package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorai/gorai/pkg/config"
)

func newTestDashboard(t *testing.T, robotCfg *config.RDL) *Dashboard {
	t.Helper()
	d, err := New(
		&config.DashboardConfig{Listen: "127.0.0.1:0"},
		robotCfg,
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return d
}

func TestHandleIndexTitleIncludesRobotName(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "my-test-robot"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "<title>my-test-robot - Gorai Dashboard</title>") {
		t.Errorf("title does not include robot name, got: %s", body[:200])
	}
}

func TestHandleIndexNavBrandIncludesRobotName(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "my-test-robot"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Gorai - my-test-robot") {
		t.Errorf("nav brand does not include robot name")
	}
}

func TestHandleIndexShowsServicesSummary(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "test-robot"},
		Services: []config.ServiceConfig{
			{Name: "suntimes", Type: "astronomical", Model: "suntimes"},
			{Name: "historian", Type: "historian", Model: "victoriametrics"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "<h3>Services</h3>") {
		t.Error("services summary card missing")
	}
	if !strings.Contains(body, ">2<") {
		t.Error("services count should be 2")
	}
}

func TestHandleIndexShowsServicesSection(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "test-robot"},
		Services: []config.ServiceConfig{
			{Name: "suntimes", Type: "astronomical", Model: "suntimes"},
			{Name: "historian", Type: "historian", Model: "victoriametrics", Disabled: true},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "suntimes") {
		t.Error("suntimes service not found in body")
	}
	if !strings.Contains(body, "astronomical / suntimes") {
		t.Error("suntimes type/model not shown")
	}
	if !strings.Contains(body, "historian / victoriametrics") {
		t.Error("historian type/model not shown")
	}
}

func TestHandleIndexNoCamerasSummaryCard(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "test-robot"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()

	// The summary grid should not have a "Cameras" card
	if strings.Contains(body, "<h3>Cameras</h3>") {
		t.Error("cameras summary card should not exist in summary grid")
	}
}

func TestHandleIndexComponentsListed(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "test-robot"},
		Components: []config.ComponentConfig{
			{Name: "solar_inverter", Type: "power_meter", Model: "growatt"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "solar_inverter") {
		t.Error("solar_inverter component not listed")
	}
	if !strings.Contains(body, "power_meter / growatt") {
		t.Error("component type/model not shown")
	}
}

func TestHandleIndexHTMLEscapesRobotName(t *testing.T) {
	d := newTestDashboard(t, &config.RDL{
		Robot: config.RobotConfig{Name: "<script>alert(1)</script>"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	d.handleIndex(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<script>") {
		t.Error("robot name was not HTML-escaped, XSS vulnerability")
	}
}
