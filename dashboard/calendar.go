package dashboard

import (
	"fmt"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/startower-observability/blackcat/scheduler"
)

const maxOccurrences = 60

func NextOccurrences(cronSpec string, after time.Time, n int) ([]time.Time, error) {
	if cronSpec == "" {
		return nil, fmt.Errorf("dashboard: cron spec cannot be empty")
	}

	parser := cron.NewParser(
		cron.SecondOptional |
			cron.Minute |
			cron.Hour |
			cron.Dom |
			cron.Month |
			cron.Dow |
			cron.Descriptor,
	)

	schedule, err := parser.Parse(cronSpec)
	if err != nil {
		return nil, err
	}

	if n > maxOccurrences {
		n = maxOccurrences
	}
	if n <= 0 {
		return []time.Time{}, nil
	}

	occurrences := make([]time.Time, 0, n)
	cursor := after
	for range n {
		next := schedule.Next(cursor)
		occurrences = append(occurrences, next)
		cursor = next
	}

	return occurrences, nil
}

func IsHighFrequency(cronSpec string, threshold time.Duration) bool {
	occurrences, err := NextOccurrences(cronSpec, time.Now().UTC(), 3)
	if err != nil || len(occurrences) < 3 {
		return false
	}

	firstInterval := occurrences[1].Sub(occurrences[0])
	secondInterval := occurrences[2].Sub(occurrences[1])
	averageInterval := (firstInterval + secondInterval) / 2

	return averageInterval < threshold
}

// CalendarEvent represents a task execution or projected occurrence on a calendar day.
type CalendarEvent struct {
	Name        string
	Status      string // "ok", "failed", "running", "scheduled"
	Time        time.Time
	IsProjected bool // true for future cron projections
	IsHighFreq  bool // true if the job runs more often than 1/hour
}

// DayCell represents a single calendar day cell.
type DayCell struct {
	Date           time.Time
	IsCurrentMonth bool
	IsToday        bool
	Events         []CalendarEvent
	HeartbeatOK    *bool // nil if no heartbeat, true/false otherwise
}

// WeekRow is a row of 7 DayCells (Sun-Sat or Mon-Sun, always 7).
type WeekRow struct {
	Days [7]DayCell
}

// MonthGrid is the full calendar month grid.
type MonthGrid struct {
	Year  int
	Month time.Month
	Weeks []WeekRow
}

func BuildMonthGrid(
	year int,
	month time.Month,
	tasks []scheduler.TaskState,
	heartbeats []scheduler.HeartbeatResult,
	jobs []CalendarJobInfo,
) MonthGrid {
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)
	gridStart := monthStart.AddDate(0, 0, -int(monthStart.Weekday()))
	gridEnd := monthEnd.AddDate(0, 0, 6-int(monthEnd.Weekday()))

	todayUTC := time.Now().UTC()
	todayDate := time.Date(todayUTC.Year(), todayUTC.Month(), todayUTC.Day(), 0, 0, 0, 0, time.UTC)

	eventsByDay := make(map[int][]CalendarEvent)
	highFreqByName := make(map[string]bool, len(jobs))

	for _, job := range jobs {
		if !job.Enabled {
			continue
		}

		isHighFreq := IsHighFrequency(job.Schedule, time.Hour)
		highFreqByName[job.Name] = isHighFreq

		if isHighFreq {
			markerStart := monthStart
			if todayDate.After(markerStart) {
				markerStart = todayDate
			}

			for day := markerStart; !day.After(monthEnd); day = day.AddDate(0, 0, 1) {
				key := dateKey(day)
				eventsByDay[key] = append(eventsByDay[key], CalendarEvent{
					Name:        job.Name,
					Status:      "scheduled",
					Time:        day,
					IsProjected: true,
					IsHighFreq:  true,
				})
			}

			continue
		}

		occurrences, err := NextOccurrences(job.Schedule, todayUTC, maxOccurrences)
		if err != nil {
			continue
		}

		for _, occurrence := range occurrences {
			occ := occurrence.UTC()
			if occ.Year() != year || occ.Month() != month {
				continue
			}

			key := dateKey(occ)
			eventsByDay[key] = append(eventsByDay[key], CalendarEvent{
				Name:        job.Name,
				Status:      "scheduled",
				Time:        occ,
				IsProjected: true,
				IsHighFreq:  false,
			})
		}
	}

	for _, task := range tasks {
		if task.LastRun.IsZero() {
			continue
		}

		lastRun := task.LastRun.UTC()
		if lastRun.Year() != year || lastRun.Month() != month {
			continue
		}

		key := dateKey(lastRun)
		eventsByDay[key] = append(eventsByDay[key], CalendarEvent{
			Name:        task.Name,
			Status:      task.LastStatus,
			Time:        lastRun,
			IsProjected: false,
			IsHighFreq:  highFreqByName[task.Name],
		})
	}

	lastHeartbeatByDay := make(map[int]time.Time)
	heartbeatOKByDay := make(map[int]bool)
	for _, result := range heartbeats {
		ts := result.Timestamp.UTC()
		key := dateKey(ts)
		prevTS, exists := lastHeartbeatByDay[key]
		if !exists || ts.After(prevTS) {
			lastHeartbeatByDay[key] = ts
			heartbeatOKByDay[key] = result.OverallHealthy
		}
	}

	weeks := make([]WeekRow, 0)
	current := gridStart
	for !current.After(gridEnd) {
		var week WeekRow
		for dayIndex := 0; dayIndex < 7; dayIndex++ {
			date := current.UTC()
			date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
			key := dateKey(date)

			dayEvents := append([]CalendarEvent(nil), eventsByDay[key]...)
			sort.Slice(dayEvents, func(i, j int) bool {
				if dayEvents[i].Time.Equal(dayEvents[j].Time) {
					if dayEvents[i].Name == dayEvents[j].Name {
						return dayEvents[i].Status < dayEvents[j].Status
					}
					return dayEvents[i].Name < dayEvents[j].Name
				}
				return dayEvents[i].Time.Before(dayEvents[j].Time)
			})

			var heartbeatPtr *bool
			if value, ok := heartbeatOKByDay[key]; ok {
				heartbeatValue := value
				heartbeatPtr = &heartbeatValue
			}

			week.Days[dayIndex] = DayCell{
				Date:           date,
				IsCurrentMonth: date.Month() == month,
				IsToday:        date.Equal(todayDate),
				Events:         dayEvents,
				HeartbeatOK:    heartbeatPtr,
			}

			current = current.AddDate(0, 0, 1)
		}

		weeks = append(weeks, week)
	}

	return MonthGrid{
		Year:  year,
		Month: month,
		Weeks: weeks,
	}
}

func dateKey(ts time.Time) int {
	date := ts.UTC()
	return date.Year()*10000 + int(date.Month())*100 + date.Day()
}

// MonthGridToView converts a MonthGrid data structure to a ScheduleView for template rendering.
func MonthGridToView(grid MonthGrid, now time.Time) ScheduleView {
	view := ScheduleView{
		Year:      grid.Year,
		Month:     int(grid.Month),
		MonthName: grid.Month.String(),
		Weeks:     make([]WeekView, len(grid.Weeks)),
	}

	// Convert each week
	for i, weekRow := range grid.Weeks {
		weekView := WeekView{}
		for j, dayCell := range weekRow.Days {
			dayView := DayView{
				DayNum:         dayCell.Date.Day(),
				DateStr:        dayCell.Date.Format("2006-01-02"),
				IsCurrentMonth: dayCell.IsCurrentMonth,
				IsToday:        dayCell.IsToday,
				HeartbeatOK:    dayCell.HeartbeatOK,
				Events:         make([]EventView, len(dayCell.Events)),
			}

			// Convert events for this day
			for k, event := range dayCell.Events {
				timeStr := ""
				if !event.Time.IsZero() {
					timeStr = event.Time.Format("15:04")
				}
				dayView.Events[k] = EventView{
					Name:        event.Name,
					Status:      event.Status,
					TimeStr:     timeStr,
					IsProjected: event.IsProjected,
					IsHighFreq:  event.IsHighFreq,
				}
			}

			weekView.Days[j] = dayView
		}
		view.Weeks[i] = weekView
	}

	// Calculate prev/next month navigation
	prevMonth := grid.Month - 1
	prevYear := grid.Year
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}

	nextMonth := grid.Month + 1
	nextYear := grid.Year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	view.PrevYear = prevYear
	view.PrevMonth = int(prevMonth)
	view.NextYear = nextYear
	view.NextMonth = int(nextMonth)

	return view
}
