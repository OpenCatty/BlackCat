package orchestrator

import (
	"testing"

	"github.com/startower-observability/blackcat/agent"
)

func TestRouterProfileMatch(t *testing.T) {
	profiles := map[string]*agent.Profile{
		"reviewer": {
			Name: "reviewer",
			Tags: []string{"review"},
		},
	}

	router := NewRouter(profiles)
	profileName, err := router.Route("please do a code review", profiles)
	if err != nil {
		t.Fatalf("Route() error = %v, want nil", err)
	}
	if profileName != "reviewer" {
		t.Fatalf("Route() profileName = %q, want %q", profileName, "reviewer")
	}
}

func TestRouterDefaultFallback(t *testing.T) {
	profiles := map[string]*agent.Profile{
		"planner": {
			Name: "planner",
			Tags: []string{"planning"},
		},
	}

	router := NewRouter(profiles)
	profileName, err := router.Route("unrelated task xyz", profiles)
	if err != nil {
		t.Fatalf("Route() error = %v, want nil", err)
	}
	if profileName != "" {
		t.Fatalf("Route() profileName = %q, want empty string", profileName)
	}
}

func TestRouterCaseInsensitive(t *testing.T) {
	profiles := map[string]*agent.Profile{
		"reviewer": {
			Name: "reviewer",
			Tags: []string{"ReViEw"},
		},
	}

	router := NewRouter(profiles)
	profileName, err := router.Route("PLEASE DO A CODE review", profiles)
	if err != nil {
		t.Fatalf("Route() error = %v, want nil", err)
	}
	if profileName != "reviewer" {
		t.Fatalf("Route() profileName = %q, want %q", profileName, "reviewer")
	}
}

func TestRouterMultipleProfiles(t *testing.T) {
	profiles := map[string]*agent.Profile{
		"reviewer": {
			Name: "reviewer",
			Tags: []string{"review"},
		},
		"planner": {
			Name: "planner",
			Tags: []string{"plan"},
		},
		"debugger": {
			Name: "debugger",
			Tags: []string{"debug"},
		},
	}

	router := NewRouter(profiles)
	profileName, err := router.Route("can you help me debug this crash", profiles)
	if err != nil {
		t.Fatalf("Route() error = %v, want nil", err)
	}
	if profileName != "debugger" {
		t.Fatalf("Route() profileName = %q, want %q", profileName, "debugger")
	}
}
