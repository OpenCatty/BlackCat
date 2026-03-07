package skills

import (
	"reflect"
	"testing"
)

func TestParseFrontmatter_ValidFrontmatter(t *testing.T) {
	content := `---
name: Test Skill
description: A test skill
tags:
  - testing
  - example
---
This is the body content.
It can span multiple lines.`

	data, body, has := ParseFrontmatter(content)

	if !has {
		t.Error("Expected hasFrontmatter to be true")
	}

	if data.Name != "Test Skill" {
		t.Errorf("Expected name 'Test Skill', got %q", data.Name)
	}

	if data.Description != "A test skill" {
		t.Errorf("Expected description 'A test skill', got %q", data.Description)
	}

	if len(data.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(data.Tags))
	}

	if data.Tags[0] != "testing" || data.Tags[1] != "example" {
		t.Errorf("Expected tags ['testing', 'example'], got %v", data.Tags)
	}

	expectedBody := "This is the body content.\nIt can span multiple lines."
	if body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, body)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := `# Coding Best Practices

**Tags**: coding, development

When writing code...`

	data, body, has := ParseFrontmatter(content)

	if has {
		t.Error("Expected hasFrontmatter to be false")
	}

	if body != content {
		t.Errorf("Expected body to be unchanged, got %q", body)
	}

	// data should be zero value
	if data.Name != "" || data.Description != "" || len(data.Tags) != 0 {
		t.Errorf("Expected zero-value FrontmatterData, got %+v", data)
	}
}

func TestParseFrontmatter_MalformedYAML(t *testing.T) {
	content := `---
name: Test Skill
invalid yaml: [
---
This is the body.`

	data, body, has := ParseFrontmatter(content)

	if has {
		t.Error("Expected hasFrontmatter to be false for malformed YAML")
	}

	if body != content {
		t.Errorf("Expected body to be unchanged after malformed YAML, got %q", body)
	}

	// data should be zero value
	if data.Name != "" {
		t.Errorf("Expected zero-value FrontmatterData, got %+v", data)
	}
}

func TestParseFrontmatter_WithMCP(t *testing.T) {
	content := `---
name: MCP Skill
description: Skill with MCP config
mcp:
  myserver:
    command: /usr/bin/myserver
    args:
      - --verbose
      - --port=8000
    env:
      MY_VAR: value
---
Body here.`

	data, _, has := ParseFrontmatter(content)

	if !has {
		t.Error("Expected hasFrontmatter to be true")
	}

	if data.Name != "MCP Skill" {
		t.Errorf("Expected name 'MCP Skill', got %q", data.Name)
	}

	if _, ok := data.MCP["myserver"]; !ok {
		t.Error("Expected MCP config for 'myserver' not found")
	}

	mcpServer := data.MCP["myserver"]
	if mcpServer.Command != "/usr/bin/myserver" {
		t.Errorf("Expected command '/usr/bin/myserver', got %q", mcpServer.Command)
	}

	if len(mcpServer.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(mcpServer.Args))
	}

	if mcpServer.Env["MY_VAR"] != "value" {
		t.Errorf("Expected env MY_VAR='value', got %q", mcpServer.Env["MY_VAR"])
	}
}

func TestParseFrontmatter_DotDotDotDelimiter(t *testing.T) {
	content := `---
name: Test Skill
...
This is the body.`

	data, body, has := ParseFrontmatter(content)

	if !has {
		t.Error("Expected hasFrontmatter to be true with ... delimiter")
	}

	if data.Name != "Test Skill" {
		t.Errorf("Expected name 'Test Skill', got %q", data.Name)
	}

	expectedBody := "This is the body."
	if body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, body)
	}
}

func TestParseFrontmatter_EmptyBody(t *testing.T) {
	content := `---
name: Empty Body Skill
---`

	data, body, has := ParseFrontmatter(content)

	if !has {
		t.Error("Expected hasFrontmatter to be true")
	}

	if data.Name != "Empty Body Skill" {
		t.Errorf("Expected name 'Empty Body Skill', got %q", data.Name)
	}

	if body != "" {
		t.Errorf("Expected empty body, got %q", body)
	}
}

func TestParseFrontmatterPhase3Fields(t *testing.T) {
	cases := []struct {
		name          string
		content       string
		wantVersion   string
		wantInstall   string
		wantDependsOn []string
		wantAnyBins   [][]string
		checkDepsNil  bool // true = assert DependsOn is nil
	}{
		{
			name: "all Phase 3 fields present",
			content: `---
name: test-skill
version: v1.2.3
install: brew install jq
depends_on:
  - base-skill
  - utils-skill
requires:
  any_bins:
    - [jq, jq-osx-amd64]
    - [curl]
---
body content`,
			wantVersion:   "v1.2.3",
			wantInstall:   "brew install jq",
			wantDependsOn: []string{"base-skill", "utils-skill"},
			wantAnyBins:   [][]string{{"jq", "jq-osx-amd64"}, {"curl"}},
		},
		{
			name: "version only no install depends_on any_bins",
			content: `---
name: minimal
version: v0.1.0
---
body`,
			wantVersion:   "v0.1.0",
			wantInstall:   "",
			wantDependsOn: nil,
			wantAnyBins:   nil,
			checkDepsNil:  true,
		},
		{
			name: "any_bins single group",
			content: `---
name: bins-only
requires:
  any_bins:
    - [python3, python]
---`,
			wantVersion:   "",
			wantInstall:   "",
			wantDependsOn: nil,
			wantAnyBins:   [][]string{{"python3", "python"}},
			checkDepsNil:  true,
		},
		{
			name: "empty new fields legacy skill",
			content: `---
name: legacy
description: old skill
---
body`,
			wantVersion:   "",
			wantInstall:   "",
			wantDependsOn: nil,
			wantAnyBins:   nil,
			checkDepsNil:  true,
		},
		{
			name: "depends_on empty explicit list",
			content: `---
name: no-deps
depends_on: []
---`,
			wantVersion:   "",
			wantInstall:   "",
			wantDependsOn: []string{},
			wantAnyBins:   nil,
		},
		{
			name: "install with special characters",
			content: `---
name: complex-install
install: 'pip install "my-package[extra]" --user'
---`,
			wantVersion:   "",
			wantInstall:   `pip install "my-package[extra]" --user`,
			wantDependsOn: nil,
			wantAnyBins:   nil,
			checkDepsNil:  true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data, _, has := ParseFrontmatter(tc.content)

			if !has {
				t.Fatal("Expected hasFrontmatter to be true")
			}

			if data.Version != tc.wantVersion {
				t.Errorf("Version: want %q, got %q", tc.wantVersion, data.Version)
			}

			if data.Install != tc.wantInstall {
				t.Errorf("Install: want %q, got %q", tc.wantInstall, data.Install)
			}

			if tc.checkDepsNil {
				if data.DependsOn != nil {
					t.Errorf("DependsOn: want nil, got %v", data.DependsOn)
				}
			} else {
				if !reflect.DeepEqual(data.DependsOn, tc.wantDependsOn) {
					t.Errorf("DependsOn: want %v, got %v", tc.wantDependsOn, data.DependsOn)
				}
			}

			if !reflect.DeepEqual(data.Requires.AnyBins, tc.wantAnyBins) {
				t.Errorf("AnyBins: want %v, got %v", tc.wantAnyBins, data.Requires.AnyBins)
			}
		})
	}
}
