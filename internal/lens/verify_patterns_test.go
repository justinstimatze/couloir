package lens

import "testing"

func TestMatchVerificationCommand(t *testing.T) {
	cases := []struct {
		command string
		want    string
	}{
		{"go test ./...", "test"},
		{"npm run test -- --watch=false", "test"},
		{"golangci-lint run ./...", "lint"},
		{"go vet ./...", "lint"},
		{"tsc --noEmit", "typecheck"},
		{"docker build -t foo .", "build"},
		{"ls -la", ""},
		{"echo hello world", ""},
	}
	for _, c := range cases {
		if got := matchVerificationCommand(c.command); got != c.want {
			t.Errorf("matchVerificationCommand(%q) = %q, want %q", c.command, got, c.want)
		}
	}
}

func TestIsBrowserOrE2ETool(t *testing.T) {
	if !isBrowserOrE2ETool("mcp__claude-in-chrome__navigate") {
		t.Error("expected mcp__claude-in-chrome__navigate to match")
	}
	if isBrowserOrE2ETool("Bash") {
		t.Error("expected Bash not to match")
	}
}
