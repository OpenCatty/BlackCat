package agent

import (
	"strings"
	"testing"
)

// --- IsAmbiguous tests ---

func TestIsAmbiguous_EmptyMessage(t *testing.T) {
	if !IsAmbiguous("") {
		t.Error("empty message should be ambiguous")
	}
	if !IsAmbiguous("   ") {
		t.Error("whitespace-only message should be ambiguous")
	}
}

func TestIsAmbiguous_VeryShort_NotGreeting(t *testing.T) {
	cases := []string{"do", "xyz", "abc", "fix", "go"}
	for _, c := range cases {
		if !IsAmbiguous(c) {
			t.Errorf("short non-greeting %q should be ambiguous", c)
		}
	}
}

func TestIsAmbiguous_ShortGreetings_NotAmbiguous(t *testing.T) {
	cases := []string{"hello", "hi", "hey", "status", "help", "ping", "yes", "no", "ok", "stop", "restart"}
	for _, c := range cases {
		if IsAmbiguous(c) {
			t.Errorf("greeting/command %q should NOT be ambiguous", c)
		}
	}
}

func TestIsAmbiguous_VagueKeywords(t *testing.T) {
	cases := []string{
		"do something with the code",
		"fix everything in the project",
		"help me with whatever this is",
		"make it work somehow",
		"I need help with stuff",
		"do the thing for me please",
		"can you do some kind of update",
		"just do anything that helps",
	}
	for _, c := range cases {
		if !IsAmbiguous(c) {
			t.Errorf("vague message %q should be ambiguous", c)
		}
	}
}

func TestIsAmbiguous_PronounPhrases(t *testing.T) {
	cases := []string{
		"fix it",
		"change it",
		"run it",
		"delete it",
		"do that",
		"update it please",
		"can you fix it",
	}
	for _, c := range cases {
		if !IsAmbiguous(c) {
			t.Errorf("pronoun phrase %q should be ambiguous", c)
		}
	}
}

func TestIsAmbiguous_ClearRequests_NotAmbiguous(t *testing.T) {
	cases := []string{
		"Create a new file called main.go with a hello world program",
		"Search for all uses of the Logger interface in the codebase",
		"Run the test suite in internal/agent",
		"Deploy the application to staging environment",
		"Refactor the handleMessage function to use error wrapping",
		"What is the current Go version in go.mod?",
		"List all files in the internal/memory directory",
		"Add a new endpoint /api/health to the HTTP server",
	}
	for _, c := range cases {
		if IsAmbiguous(c) {
			t.Errorf("clear request %q should NOT be ambiguous", c)
		}
	}
}

func TestIsAmbiguous_CaseInsensitive(t *testing.T) {
	if !IsAmbiguous("DO SOMETHING with this") {
		t.Error("case-insensitive keyword match should work")
	}
	if !IsAmbiguous("Make It Work") {
		t.Error("case-insensitive keyword match should work for 'make it work'")
	}
}

// --- ClarificationPromptSection tests ---

func TestClarificationPromptSection_AmbiguousMessage(t *testing.T) {
	section := ClarificationPromptSection("fix it")
	if section == "" {
		t.Fatal("expected non-empty clarification section for ambiguous message")
	}
	if !strings.Contains(section, "Clarification Required") {
		t.Error("section should contain 'Clarification Required' header")
	}
	if !strings.Contains(section, "clarifying questions") {
		t.Error("section should mention clarifying questions")
	}
	if !strings.Contains(section, "Do NOT execute tools") {
		t.Error("section should instruct not to execute tools")
	}
}

func TestClarificationPromptSection_ClearMessage_Empty(t *testing.T) {
	section := ClarificationPromptSection("Create a new file called server.go with an HTTP handler")
	if section != "" {
		t.Errorf("expected empty section for clear message, got %q", section)
	}
}

func TestClarificationPromptSection_EmptyMessage(t *testing.T) {
	section := ClarificationPromptSection("")
	if section == "" {
		t.Fatal("expected non-empty section for empty message")
	}
}

func TestClarificationPromptSection_Greeting_Empty(t *testing.T) {
	section := ClarificationPromptSection("hello")
	if section != "" {
		t.Errorf("expected empty section for greeting, got %q", section)
	}
}
