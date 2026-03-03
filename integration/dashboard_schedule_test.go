package integration_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/config"
	"github.com/startower-observability/blackcat/dashboard"
	"github.com/startower-observability/blackcat/scheduler"
)

// dashboardMockTaskDetailLister implements dashboard.TaskDetailLister
type dashboardMockTaskDetailLister struct {
	tasks []scheduler.TaskState
}

func (m dashboardMockTaskDetailLister) ListTasks() []scheduler.TaskState {
	return append([]scheduler.TaskState(nil), m.tasks...)
}

// dashboardMockScheduleProvider implements dashboard.ScheduleProvider
type dashboardMockScheduleProvider struct {
	jobs []dashboard.CalendarJobInfo
}

func (m dashboardMockScheduleProvider) ListJobs() []dashboard.CalendarJobInfo {
	return append([]dashboard.CalendarJobInfo(nil), m.jobs...)
}

func TestDashboardSchedulePage(t *testing.T) {
	t.Parallel()

	deps := dashboard.DashboardDeps{
		TaskDetailLister: dashboardMockTaskDetailLister{},
		ScheduleProvider: dashboardMockScheduleProvider{},
	}
	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, deps)
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	res := mustRequest(t, ts.URL+"/dashboard/schedule", "secret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("expected text/html, got %q", contentType)
	}
	body := mustReadBody(t, res)
	// calendar-container should be present
	if !strings.Contains(body, "calendar-container") {
		t.Fatalf("expected calendar-container in body, got %q", body[:min(200, len(body))])
	}
}

func TestDashboardSchedulePageMonthNav(t *testing.T) {
	t.Parallel()

	deps := dashboard.DashboardDeps{
		TaskDetailLister: dashboardMockTaskDetailLister{},
		ScheduleProvider: dashboardMockScheduleProvider{},
	}
	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, deps)
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	// Navigate to March 2025
	res := mustRequest(t, ts.URL+"/dashboard/schedule?year=2025&month=3", "secret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
	body := mustReadBody(t, res)
	if !strings.Contains(body, "March") {
		t.Fatalf("expected 'March' in body for month=3, got %q", body[:min(500, len(body))])
	}
	if !strings.Contains(body, "2025") {
		t.Fatalf("expected '2025' in body, got %q", body[:min(500, len(body))])
	}
}

func TestDashboardScheduleAPIEndpoint(t *testing.T) {
	t.Parallel()

	deps := dashboard.DashboardDeps{
		TaskDetailLister: dashboardMockTaskDetailLister{},
		ScheduleProvider: dashboardMockScheduleProvider{
			jobs: []dashboard.CalendarJobInfo{
				{Name: "heartbeat", Schedule: "@every 30s", Enabled: true},
			},
		},
	}
	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, deps)
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	res := mustRequest(t, ts.URL+"/dashboard/api/schedule?year=2025&month=3", "secret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json, got %q", contentType)
	}

	var payload struct {
		Year      int    `json:"year"`
		Month     int    `json:"month"`
		MonthName string `json:"month_name"`
		Weeks     []struct {
			Days [7]struct {
				DayNum         int  `json:"day_num"`
				IsCurrentMonth bool `json:"is_current_month"`
			} `json:"days"`
		} `json:"weeks"`
		PrevYear  int `json:"prev_year"`
		PrevMonth int `json:"prev_month"`
		NextYear  int `json:"next_year"`
		NextMonth int `json:"next_month"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if payload.Year != 2025 {
		t.Fatalf("expected year=2025, got %d", payload.Year)
	}
	if payload.Month != 3 {
		t.Fatalf("expected month=3, got %d", payload.Month)
	}
	if payload.MonthName != "March" {
		t.Fatalf("expected month_name='March', got %q", payload.MonthName)
	}
	if len(payload.Weeks) == 0 {
		t.Fatal("expected non-empty weeks")
	}
	// March 2025 should have at least 5 weeks
	if len(payload.Weeks) < 5 {
		t.Fatalf("expected at least 5 weeks for March 2025, got %d", len(payload.Weeks))
	}
	// Navigation: Feb 2025 prev, Apr 2025 next
	if payload.PrevMonth != 2 || payload.PrevYear != 2025 {
		t.Fatalf("expected prev=2025/2, got %d/%d", payload.PrevYear, payload.PrevMonth)
	}
	if payload.NextMonth != 4 || payload.NextYear != 2025 {
		t.Fatalf("expected next=2025/4, got %d/%d", payload.NextYear, payload.NextMonth)
	}
}
