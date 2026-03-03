package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/startower-observability/blackcat/internal/daemon"
	"github.com/startower-observability/blackcat/internal/scheduler"
)

var Version = "dev"

type APIHandler struct {
	subsystemManager SubsystemManager
	scheduler        TaskLister
	heartbeatStore   HeartbeatStore
	taskDetailLister TaskDetailLister
	scheduleProvider ScheduleProvider
	renderer         *TemplateRenderer
	startupTime      time.Time
}

type apiStatusResponse struct {
	Uptime         string `json:"uptime"`
	Version        string `json:"version"`
	Healthy        bool   `json:"healthy"`
	SubsystemCount int    `json:"subsystem_count"`
}

type apiAgentResponse struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	LastActive string `json:"last_active"`
}

type apiTaskResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type apiTasksResponse struct {
	Tasks []apiTaskResponse `json:"tasks"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

type apiEventResponse struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	TimeStr     string `json:"time_str"`
	IsProjected bool   `json:"is_projected"`
	IsHighFreq  bool   `json:"is_high_freq"`
}

type apiDayResponse struct {
	DayNum         int                `json:"day_num"`
	DateStr        string             `json:"date_str"`
	IsCurrentMonth bool               `json:"is_current_month"`
	IsToday        bool               `json:"is_today"`
	Events         []apiEventResponse `json:"events"`
	HeartbeatOK    *bool              `json:"heartbeat_ok"`
}

type apiWeekResponse struct {
	Days [7]apiDayResponse `json:"days"`
}

type apiScheduleResponse struct {
	Year      int               `json:"year"`
	Month     int               `json:"month"`
	MonthName string            `json:"month_name"`
	Weeks     []apiWeekResponse `json:"weeks"`
	PrevYear  int               `json:"prev_year"`
	PrevMonth int               `json:"prev_month"`
	NextYear  int               `json:"next_year"`
	NextMonth int               `json:"next_month"`
}

func NewAPIHandler(deps DashboardDeps, startupTime time.Time) *APIHandler {
	renderer, err := NewTemplateRenderer()
	if err != nil {
		renderer = nil
	}

	return &APIHandler{
		subsystemManager: deps.SubsystemManager,
		scheduler:        deps.TaskLister,
		heartbeatStore:   deps.HeartbeatStore,
		taskDetailLister: deps.TaskDetailLister,
		scheduleProvider: deps.ScheduleProvider,
		renderer:         renderer,
		startupTime:      startupTime,
	}
}

func (h *APIHandler) RegisterRoutes(r chi.Router) {
	r.Get("/schedule", h.handleSchedule)
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", h.handleStatus)
		r.Get("/agents", h.handleAgents)
		r.Get("/tasks", h.handleTasks)
		r.Get("/tasks/{id}", h.handleTaskDetail)
		r.Get("/health", h.handleHealth)
		r.Get("/schedule", h.handleAPISchedule)
	})
}

func (h *APIHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	subsystems := h.listSubsystems()
	status := apiStatusResponse{
		Uptime:         h.uptime(),
		Version:        Version,
		Healthy:        subsystemsHealthy(subsystems),
		SubsystemCount: len(subsystems),
	}

	if prefersHTML(r) {
		state := "failed"
		if status.Healthy {
			state = "healthy"
		}

		view := AgentView{
			Name:        "blackcat",
			State:       state,
			CurrentTask: "version " + status.Version,
			LastActive:  status.Uptime,
		}

		if h.renderAgentCards(w, http.StatusOK, []AgentView{view}) {
			return
		}
	}

	writeJSON(w, http.StatusOK, status)
}

func (h *APIHandler) handleAgents(w http.ResponseWriter, r *http.Request) {
	subsystems := h.listSubsystems()
	agents := make([]apiAgentResponse, 0, len(subsystems))
	views := make([]AgentView, 0, len(subsystems))

	now := time.Now().UTC().Format(time.RFC3339)
	for _, subsystem := range subsystems {
		name := strings.TrimSpace(subsystem.Name)
		if name == "" {
			name = "unknown"
		}

		state := normalizeSubsystemState(subsystem.Status)
		agents = append(agents, apiAgentResponse{
			Name:       name,
			State:      state,
			LastActive: now,
		})

		views = append(views, AgentView{
			Name:        name,
			State:       state,
			CurrentTask: subsystem.Message,
			LastActive:  now,
		})
	}

	if prefersHTML(r) && h.renderAgentCards(w, http.StatusOK, views) {
		return
	}

	writeJSON(w, http.StatusOK, agents)
}

func (h *APIHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)

	tasks := h.listTaskResponses()
	total := len(tasks)
	start := (page - 1) * limit
	if start > total {
		start = total
	}

	end := start + limit
	if end > total {
		end = total
	}

	pagedTasks := make([]apiTaskResponse, 0, end-start)
	if start < end {
		pagedTasks = append(pagedTasks, tasks[start:end]...)
	}

	response := apiTasksResponse{
		Tasks: pagedTasks,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	if prefersHTML(r) {
		rows := make([]TaskView, 0, len(pagedTasks))
		for _, task := range pagedTasks {
			rows = append(rows, taskToView(task))
		}

		if h.renderTaskRows(w, http.StatusOK, rows) {
			return
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *APIHandler) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimSpace(chi.URLParam(r, "id"))
	task, found := h.findTask(taskID)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	if prefersHTML(r) && h.renderTaskRows(w, http.StatusOK, []TaskView{taskToView(task)}) {
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	results := make([]scheduler.HeartbeatResult, 0)
	if h.heartbeatStore != nil {
		results = h.heartbeatStore.Latest(10)
	}

	if prefersHTML(r) {
		agents := make([]AgentView, 0, len(results))
		for i, result := range results {
			state := "failed"
			if result.OverallHealthy {
				state = "healthy"
			}

			lastActive := "unknown"
			if !result.Timestamp.IsZero() {
				lastActive = result.Timestamp.UTC().Format(time.RFC3339)
			}

			agents = append(agents, AgentView{
				Name:        "heartbeat-" + strconv.Itoa(i+1),
				State:       state,
				CurrentTask: "subsystems: " + strconv.Itoa(len(result.Subsystems)),
				LastActive:  lastActive,
			})
		}

		if h.renderAgentCards(w, http.StatusOK, agents) {
			return
		}
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *APIHandler) uptime() string {
	if h.startupTime.IsZero() {
		return "0s"
	}

	return time.Since(h.startupTime).Round(time.Second).String()
}

func (h *APIHandler) listSubsystems() []daemon.SubsystemHealth {
	if h.subsystemManager == nil {
		return []daemon.SubsystemHealth{}
	}

	return h.subsystemManager.Healthz()
}

func (h *APIHandler) listTaskResponses() []apiTaskResponse {
	if h.scheduler == nil {
		return []apiTaskResponse{}
	}

	taskNames := h.scheduler.ListTasks()
	tasks := make([]apiTaskResponse, 0, len(taskNames))
	for i, name := range taskNames {
		tasks = append(tasks, apiTaskResponse{
			ID:    strconv.Itoa(i + 1),
			Name:  name,
			State: "scheduled",
		})
	}

	return tasks
}

func (h *APIHandler) findTask(taskID string) (apiTaskResponse, bool) {
	tasks := h.listTaskResponses()
	if index, err := strconv.Atoi(taskID); err == nil {
		if index >= 1 && index <= len(tasks) {
			return tasks[index-1], true
		}
	}

	for _, task := range tasks {
		if task.ID == taskID || task.Name == taskID {
			return task, true
		}
	}

	return apiTaskResponse{}, false
}

func (h *APIHandler) renderAgentCards(w http.ResponseWriter, statusCode int, agents []AgentView) bool {
	if h.renderer == nil {
		return false
	}

	recorder := httptest.NewRecorder()
	for _, agent := range agents {
		if err := h.renderer.RenderPartial(recorder, "agent-card", agent); err != nil {
			return false
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(recorder.Body.Bytes())
	return true
}

func (h *APIHandler) renderTaskRows(w http.ResponseWriter, statusCode int, rows []TaskView) bool {
	if h.renderer == nil {
		return false
	}

	recorder := httptest.NewRecorder()
	for _, row := range rows {
		if err := h.renderer.RenderPartial(recorder, "task-row", row); err != nil {
			return false
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(recorder.Body.Bytes())
	return true
}

func taskToView(task apiTaskResponse) TaskView {
	return TaskView{
		ID:       task.ID,
		Name:     task.Name,
		Status:   task.State,
		Duration: "-",
		LastRun:  "-",
	}
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func prefersHTML(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/html")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func normalizeSubsystemState(status string) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	if normalized == "" {
		return "unknown"
	}

	return normalized
}

func subsystemsHealthy(subsystems []daemon.SubsystemHealth) bool {
	for _, subsystem := range subsystems {
		switch normalizeSubsystemState(subsystem.Status) {
		case "running", "healthy", "ok":
		default:
			return false
		}
	}

	return true
}

func (h *APIHandler) handleSchedule(w http.ResponseWriter, r *http.Request) {
	// Parse year and month from query params, default to current month
	now := time.Now().UTC()
	year := now.Year()
	month := now.Month()

	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}
	if m := r.URL.Query().Get("month"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed >= 1 && parsed <= 12 {
			month = time.Month(parsed)
		}
	}

	// Gather data
	var tasks []scheduler.TaskState
	if h.taskDetailLister != nil {
		tasks = h.taskDetailLister.ListTasks()
	}

	var heartbeats []scheduler.HeartbeatResult
	if h.heartbeatStore != nil {
		heartbeats = h.heartbeatStore.Latest(100)
	}

	var jobs []CalendarJobInfo
	if h.scheduleProvider != nil {
		jobs = h.scheduleProvider.ListJobs()
	}

	grid := BuildMonthGrid(year, month, tasks, heartbeats, jobs)
	view := MonthGridToView(grid, now)

	if h.renderer == nil {
		writeJSON(w, http.StatusOK, view)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := h.renderer.Render(w, "schedule", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func (h *APIHandler) handleAPISchedule(w http.ResponseWriter, r *http.Request) {
	// Parse year and month from query params, default to current month
	now := time.Now().UTC()
	year := now.Year()
	month := now.Month()

	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}
	if m := r.URL.Query().Get("month"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed >= 1 && parsed <= 12 {
			month = time.Month(parsed)
		}
	}

	// Gather data
	var tasks []scheduler.TaskState
	if h.taskDetailLister != nil {
		tasks = h.taskDetailLister.ListTasks()
	}

	var heartbeats []scheduler.HeartbeatResult
	if h.heartbeatStore != nil {
		heartbeats = h.heartbeatStore.Latest(100)
	}

	var jobs []CalendarJobInfo
	if h.scheduleProvider != nil {
		jobs = h.scheduleProvider.ListJobs()
	}

	grid := BuildMonthGrid(year, month, tasks, heartbeats, jobs)
	view := MonthGridToView(grid, now)

	// Always return JSON, never HTML
	writeJSON(w, http.StatusOK, scheduleViewToJSON(view))
}

func scheduleViewToJSON(v ScheduleView) apiScheduleResponse {
	// Convert weeks and days
	weeks := make([]apiWeekResponse, len(v.Weeks))
	for i, week := range v.Weeks {
		days := [7]apiDayResponse{}
		for j, day := range week.Days {
			events := make([]apiEventResponse, len(day.Events))
			for k, event := range day.Events {
				events[k] = apiEventResponse{
					Name:        event.Name,
					Status:      event.Status,
					TimeStr:     event.TimeStr,
					IsProjected: event.IsProjected,
					IsHighFreq:  event.IsHighFreq,
				}
			}
			days[j] = apiDayResponse{
				DayNum:         day.DayNum,
				DateStr:        day.DateStr,
				IsCurrentMonth: day.IsCurrentMonth,
				IsToday:        day.IsToday,
				Events:         events,
				HeartbeatOK:    day.HeartbeatOK,
			}
		}
		weeks[i] = apiWeekResponse{Days: days}
	}

	return apiScheduleResponse{
		Year:      v.Year,
		Month:     v.Month,
		MonthName: v.MonthName,
		Weeks:     weeks,
		PrevYear:  v.PrevYear,
		PrevMonth: v.PrevMonth,
		NextYear:  v.NextYear,
		NextMonth: v.NextMonth,
	}

}
