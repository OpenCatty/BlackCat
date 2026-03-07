package skills

import (
	"strings"
	"testing"
)

// TestResolveDependencies_NoDeps skills with no DependsOn → alphabetical order from Kahn's
func TestResolveDependencies_NoDeps(t *testing.T) {
	skills := []Skill{
		{Name: "b"},
		{Name: "a"},
		{Name: "c"},
	}

	result, err := ResolveDependencies(skills)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(result))
	}

	// Kahn's with sorted zero-in-degree produces alphabetical order
	expected := []string{"a", "b", "c"}
	for i, name := range expected {
		if result[i].Name != name {
			t.Fatalf("result[%d]: expected %q, got %q", i, name, result[i].Name)
		}
	}
}

// TestResolveDependencies_LinearChain A→B→C chain → result is [C, B, A]
func TestResolveDependencies_LinearChain(t *testing.T) {
	skills := []Skill{
		{Name: "A", DependsOn: []string{"B"}},
		{Name: "B", DependsOn: []string{"C"}},
		{Name: "C"},
	}

	result, err := ResolveDependencies(skills)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(result))
	}

	if result[0].Name != "C" {
		t.Fatalf("result[0]: expected \"C\", got %q", result[0].Name)
	}
	if result[1].Name != "B" {
		t.Fatalf("result[1]: expected \"B\", got %q", result[1].Name)
	}
	if result[2].Name != "A" {
		t.Fatalf("result[2]: expected \"A\", got %q", result[2].Name)
	}
}

// TestResolveDependencies_Diamond D depends on B+C, B depends on A, C depends on A
func TestResolveDependencies_Diamond(t *testing.T) {
	skills := []Skill{
		{Name: "D", DependsOn: []string{"B", "C"}},
		{Name: "B", DependsOn: []string{"A"}},
		{Name: "C", DependsOn: []string{"A"}},
		{Name: "A"},
	}

	result, err := ResolveDependencies(skills)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 skills, got %d", len(result))
	}

	// A first (only zero-in-degree), then B and C (alphabetical), then D
	expected := []string{"A", "B", "C", "D"}
	for i, name := range expected {
		if result[i].Name != name {
			t.Fatalf("result[%d]: expected %q, got %q", i, name, result[i].Name)
		}
	}
}

// TestResolveDependencies_CycleDetected A→B, B→A → returns error containing "cycle"
func TestResolveDependencies_CycleDetected(t *testing.T) {
	skills := []Skill{
		{Name: "A", DependsOn: []string{"B"}},
		{Name: "B", DependsOn: []string{"A"}},
	}

	_, err := ResolveDependencies(skills)
	if err == nil {
		t.Fatal("expected error for cycle, got nil")
	}

	errLower := strings.ToLower(err.Error())
	if !strings.Contains(errLower, "cycle") && !strings.Contains(errLower, "circular") {
		t.Fatalf("expected error to contain 'cycle' or 'circular', got: %v", err)
	}
}

// TestResolveDependencies_SelfDep A depends on itself → error (cycle)
func TestResolveDependencies_SelfDep(t *testing.T) {
	skills := []Skill{
		{Name: "A", DependsOn: []string{"A"}},
	}

	_, err := ResolveDependencies(skills)
	if err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}

	errLower := strings.ToLower(err.Error())
	if !strings.Contains(errLower, "cycle") && !strings.Contains(errLower, "circular") {
		t.Fatalf("expected error to contain 'cycle' or 'circular', got: %v", err)
	}
}

// TestResolveDependencies_MissingDep A depends on "missing" → A removed, result empty, no error
func TestResolveDependencies_MissingDep(t *testing.T) {
	skills := []Skill{
		{Name: "A", DependsOn: []string{"missing"}},
	}

	result, err := ResolveDependencies(skills)
	if err != nil {
		t.Fatalf("expected no error for missing dep, got: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 skills (A removed due to missing dep), got %d", len(result))
	}
}

// TestResolveDependencies_Mixed some skills with deps, some without, one missing dep
func TestResolveDependencies_Mixed(t *testing.T) {
	skills := []Skill{
		{Name: "standalone"},
		{Name: "dep-on-missing", DependsOn: []string{"ghost"}},
		{Name: "dep-on-standalone", DependsOn: []string{"standalone"}},
	}

	result, err := ResolveDependencies(skills)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(result))
	}

	if result[0].Name != "standalone" {
		t.Fatalf("result[0]: expected \"standalone\", got %q", result[0].Name)
	}
	if result[1].Name != "dep-on-standalone" {
		t.Fatalf("result[1]: expected \"dep-on-standalone\", got %q", result[1].Name)
	}
}
