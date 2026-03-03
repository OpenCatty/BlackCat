package dashboard

import (
	"testing"
	"time"

	"github.com/startower-observability/blackcat/internal/scheduler"
)

func TestNextOccurrences_EveryThirtySeconds(t *testing.T) {
	after := time.Date(2026, time.January, 5, 10, 0, 0, 0, time.UTC)

	occurrences, err := NextOccurrences("@every 30s", after, 5)
	if err != nil {
		t.Fatalf("NextOccurrences returned error: %v", err)
	}

	if len(occurrences) != 5 {
		t.Fatalf("expected 5 occurrences, got %d", len(occurrences))
	}

	for i := range 5 {
		expected := after.Add(time.Duration(i+1) * 30 * time.Second)
		if !occurrences[i].Equal(expected) {
			t.Fatalf("occurrence %d mismatch: expected %v, got %v", i, expected, occurrences[i])
		}
	}
}

func TestNextOccurrences_WeekdaySchedule(t *testing.T) {
	after := time.Date(2026, time.January, 5, 9, 29, 0, 0, time.UTC)

	occurrences, err := NextOccurrences("0 30 9 * * MON-FRI", after, 5)
	if err != nil {
		t.Fatalf("NextOccurrences returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.January, 5, 9, 30, 0, 0, time.UTC),
		time.Date(2026, time.January, 6, 9, 30, 0, 0, time.UTC),
		time.Date(2026, time.January, 7, 9, 30, 0, 0, time.UTC),
		time.Date(2026, time.January, 8, 9, 30, 0, 0, time.UTC),
		time.Date(2026, time.January, 9, 9, 30, 0, 0, time.UTC),
	}

	if len(occurrences) != len(expected) {
		t.Fatalf("expected %d occurrences, got %d", len(expected), len(occurrences))
	}

	for i := range expected {
		if !occurrences[i].Equal(expected[i]) {
			t.Fatalf("occurrence %d mismatch: expected %v, got %v", i, expected[i], occurrences[i])
		}
	}
}

func TestNextOccurrences_Daily(t *testing.T) {
	after := time.Date(2026, time.January, 10, 23, 0, 0, 0, time.UTC)

	occurrences, err := NextOccurrences("@daily", after, 3)
	if err != nil {
		t.Fatalf("NextOccurrences returned error: %v", err)
	}

	expected := []time.Time{
		time.Date(2026, time.January, 11, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.January, 12, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.January, 13, 0, 0, 0, 0, time.UTC),
	}

	if len(occurrences) != len(expected) {
		t.Fatalf("expected %d occurrences, got %d", len(expected), len(occurrences))
	}

	for i := range expected {
		if !occurrences[i].Equal(expected[i]) {
			t.Fatalf("occurrence %d mismatch: expected %v, got %v", i, expected[i], occurrences[i])
		}
	}
}

func TestNextOccurrences_EmptySpec(t *testing.T) {
	occurrences, err := NextOccurrences("", time.Now().UTC(), 5)
	if err == nil {
		t.Fatal("expected error for empty spec")
	}
	if occurrences != nil {
		t.Fatalf("expected nil occurrences, got %v", occurrences)
	}
}

func TestNextOccurrences_InvalidSpec(t *testing.T) {
	occurrences, err := NextOccurrences("invalid", time.Now().UTC(), 5)
	if err == nil {
		t.Fatal("expected error for invalid spec")
	}
	if occurrences != nil {
		t.Fatalf("expected nil occurrences, got %v", occurrences)
	}
}

func TestNextOccurrences_CapEnforcement(t *testing.T) {
	after := time.Date(2026, time.January, 5, 10, 0, 0, 0, time.UTC)

	occurrences, err := NextOccurrences("@every 30s", after, 100)
	if err != nil {
		t.Fatalf("NextOccurrences returned error: %v", err)
	}

	if len(occurrences) != 60 {
		t.Fatalf("expected 60 occurrences due to cap, got %d", len(occurrences))
	}
}

func TestIsHighFrequency_HighFreq(t *testing.T) {
	if !IsHighFrequency("@every 30s", 24*time.Hour) {
		t.Fatal("expected @every 30s to be high frequency")
	}
}

func TestIsHighFrequency_LowFreq(t *testing.T) {
	if IsHighFrequency("@daily", 24*time.Hour) {
		t.Fatal("expected @daily to not be high frequency")
	}
}

func TestIsHighFrequency_Invalid(t *testing.T) {
	if IsHighFrequency("invalid", 24*time.Hour) {
		t.Fatal("expected invalid spec to return false")
	}
}

func TestBuildMonthGrid_GridDimensions_MonthStartsOnSunday(t *testing.T) {
	grid := BuildMonthGrid(2024, time.September, nil, nil, nil)

	if len(grid.Weeks) != 5 {
		t.Fatalf("expected 5 weeks, got %d", len(grid.Weeks))
	}

	firstDay := grid.Weeks[0].Days[0].Date
	if !firstDay.Equal(time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected first day to be 2024-09-01, got %v", firstDay)
	}
}

func TestBuildMonthGrid_GridDimensions_MonthStartsOnWednesday(t *testing.T) {
	grid := BuildMonthGrid(2024, time.May, nil, nil, nil)

	if len(grid.Weeks) != 5 {
		t.Fatalf("expected 5 weeks, got %d", len(grid.Weeks))
	}

	firstDay := grid.Weeks[0].Days[0].Date
	if !firstDay.Equal(time.Date(2024, time.April, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected first grid day to be 2024-04-28, got %v", firstDay)
	}
}

func TestBuildMonthGrid_PaddingDaysOutsideMonthMarkedFalse(t *testing.T) {
	grid := BuildMonthGrid(2024, time.May, nil, nil, nil)

	if grid.Weeks[0].Days[0].IsCurrentMonth {
		t.Fatal("expected leading padding day to be outside current month")
	}
	if grid.Weeks[0].Days[1].IsCurrentMonth {
		t.Fatal("expected leading padding day to be outside current month")
	}
	if grid.Weeks[0].Days[2].IsCurrentMonth {
		t.Fatal("expected leading padding day to be outside current month")
	}

	lastWeek := grid.Weeks[len(grid.Weeks)-1]
	if lastWeek.Days[6].IsCurrentMonth {
		t.Fatal("expected trailing padding day to be outside current month")
	}
}

func TestBuildMonthGrid_IsTodayMarkedCorrectly(t *testing.T) {
	now := time.Now().UTC()
	grid := BuildMonthGrid(now.Year(), now.Month(), nil, nil, nil)

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayCount := 0
	for _, week := range grid.Weeks {
		for _, day := range week.Days {
			if day.IsToday {
				todayCount++
				if !day.Date.Equal(today) {
					t.Fatalf("expected IsToday day %v, got %v", today, day.Date)
				}
			}
		}
	}

	if todayCount != 1 {
		t.Fatalf("expected exactly one IsToday cell, got %d", todayCount)
	}
}

func TestBuildMonthGrid_PastTaskRunPlacedOnLastRunDay(t *testing.T) {
	tasks := []scheduler.TaskState{
		{
			Name:       "backup",
			LastRun:    time.Date(2026, time.January, 15, 14, 30, 0, 0, time.UTC),
			LastStatus: "failed",
		},
	}

	grid := BuildMonthGrid(2026, time.January, tasks, nil, nil)
	day := mustFindDayCell(t, grid, time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC))

	if len(day.Events) != 1 {
		t.Fatalf("expected one event on task run day, got %d", len(day.Events))
	}

	event := day.Events[0]
	if event.Name != "backup" {
		t.Fatalf("expected task name backup, got %q", event.Name)
	}
	if event.Status != "failed" {
		t.Fatalf("expected task status failed, got %q", event.Status)
	}
	if event.IsProjected {
		t.Fatal("expected task run event to not be projected")
	}
}

func TestBuildMonthGrid_HeartbeatUsesLastResultForDay(t *testing.T) {
	heartbeats := []scheduler.HeartbeatResult{
		{Timestamp: time.Date(2026, time.January, 20, 10, 0, 0, 0, time.UTC), OverallHealthy: false},
		{Timestamp: time.Date(2026, time.January, 20, 11, 0, 0, 0, time.UTC), OverallHealthy: true},
	}

	grid := BuildMonthGrid(2026, time.January, nil, heartbeats, nil)
	day := mustFindDayCell(t, grid, time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC))

	if day.HeartbeatOK == nil {
		t.Fatal("expected heartbeat status to be populated")
	}
	if !*day.HeartbeatOK {
		t.Fatal("expected heartbeat to reflect last result of the day")
	}
}

func TestBuildMonthGrid_HeartbeatNilWhenNoResultForDay(t *testing.T) {
	heartbeats := []scheduler.HeartbeatResult{
		{Timestamp: time.Date(2026, time.January, 19, 10, 0, 0, 0, time.UTC), OverallHealthy: true},
	}

	grid := BuildMonthGrid(2026, time.January, nil, heartbeats, nil)
	day := mustFindDayCell(t, grid, time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC))

	if day.HeartbeatOK != nil {
		t.Fatal("expected heartbeat to be nil for day without results")
	}
}

func TestBuildMonthGrid_HighFrequencyJobsUseSingleDailyMarkers(t *testing.T) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	jobs := []CalendarJobInfo{
		{Name: "stream", Schedule: "@every 30m", Enabled: true},
	}

	grid := BuildMonthGrid(now.Year(), now.Month(), nil, nil, jobs)
	markerDays := 0

	for _, week := range grid.Weeks {
		for _, day := range week.Days {
			if day.Date.Month() != now.Month() || day.Date.Before(today) {
				continue
			}

			highFreqCount := 0
			for _, event := range day.Events {
				if event.Name == "stream" {
					highFreqCount++
					if !event.IsHighFreq {
						t.Fatal("expected high-frequency marker to set IsHighFreq")
					}
					if !event.IsProjected {
						t.Fatal("expected high-frequency marker to be projected")
					}
				}
			}

			if highFreqCount != 1 {
				t.Fatalf("expected one high-frequency marker on %v, got %d", day.Date, highFreqCount)
			}
			markerDays++
		}
	}

	expectedMarkerDays := int(monthEnd.Sub(today).Hours()/24) + 1
	if markerDays != expectedMarkerDays {
		t.Fatalf("expected %d marker days, got %d", expectedMarkerDays, markerDays)
	}
}

func mustFindDayCell(t *testing.T, grid MonthGrid, date time.Time) DayCell {
	t.Helper()
	target := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	for _, week := range grid.Weeks {
		for _, day := range week.Days {
			if day.Date.Equal(target) {
				return day
			}
		}
	}

	t.Fatalf("date %v not found in grid", target)
	return DayCell{}
}

func TestMonthGridToView_EmptyGrid(t *testing.T) {
	grid := MonthGrid{
		Year:  2026,
		Month: time.January,
		Weeks: []WeekRow{},
	}
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)

	view := MonthGridToView(grid, now)

	if view.Year != 2026 {
		t.Fatalf("expected Year=2026, got %d", view.Year)
	}
	if view.Month != 1 {
		t.Fatalf("expected Month=1, got %d", view.Month)
	}
	if view.MonthName != "January" {
		t.Fatalf("expected MonthName='January', got '%s'", view.MonthName)
	}
	if len(view.Weeks) != 0 {
		t.Fatalf("expected 0 weeks, got %d", len(view.Weeks))
	}

	// Check prev/next month navigation
	if view.PrevYear != 2025 || view.PrevMonth != 12 {
		t.Fatalf("expected PrevYear=2025, PrevMonth=12, got %d, %d", view.PrevYear, view.PrevMonth)
	}
	if view.NextYear != 2026 || view.NextMonth != 2 {
		t.Fatalf("expected NextYear=2026, NextMonth=2, got %d, %d", view.NextYear, view.NextMonth)
	}
}

func TestMonthGridToView_EventTimeFormatting(t *testing.T) {
	eventTime := time.Date(2026, time.January, 15, 14, 30, 0, 0, time.UTC)
	grid := MonthGrid{
		Year:  2026,
		Month: time.January,
		Weeks: []WeekRow{
			{
				Days: [7]DayCell{
					{
						Date:           time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC),
						IsCurrentMonth: true,
						IsToday:        true,
						Events: []CalendarEvent{
							{
								Name:        "Test Event",
								Status:      "ok",
								Time:        eventTime,
								IsProjected: false,
								IsHighFreq:  false,
							},
						},
						HeartbeatOK: nil,
					},
				},
			},
		},
	}
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)

	view := MonthGridToView(grid, now)

	if len(view.Weeks) != 1 {
		t.Fatalf("expected 1 week, got %d", len(view.Weeks))
	}

	dayView := view.Weeks[0].Days[0]
	if len(dayView.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(dayView.Events))
	}

	eventView := dayView.Events[0]
	if eventView.TimeStr != "14:30" {
		t.Fatalf("expected TimeStr='14:30', got '%s'", eventView.TimeStr)
	}
	if eventView.Name != "Test Event" {
		t.Fatalf("expected Name='Test Event', got '%s'", eventView.Name)
	}
	if eventView.Status != "ok" {
		t.Fatalf("expected Status='ok', got '%s'", eventView.Status)
	}
}
