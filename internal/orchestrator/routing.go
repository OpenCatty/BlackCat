package orchestrator

import (
	"strings"

	"github.com/startower-observability/blackcat/internal/agent"
)

// Router selects a profile using task keyword matching.
type Router struct {
	profiles map[string]*agent.Profile
}

func NewRouter(profiles map[string]*agent.Profile) *Router {
	return &Router{profiles: profiles}
}

func (r *Router) Route(task string, profiles map[string]*agent.Profile) (profileName string, err error) {
	searchProfiles := profiles
	if len(searchProfiles) == 0 && r != nil {
		searchProfiles = r.profiles
	}

	taskLower := strings.ToLower(task)
	for name, profile := range searchProfiles {
		if profile == nil {
			continue
		}

		for _, tag := range profile.Tags {
			tagLower := strings.ToLower(strings.TrimSpace(tag))
			if tagLower == "" {
				continue
			}

			if strings.Contains(taskLower, tagLower) {
				return name, nil
			}
		}
	}

	return "", nil
}
