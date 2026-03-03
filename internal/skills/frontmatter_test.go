package skills

import (
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
