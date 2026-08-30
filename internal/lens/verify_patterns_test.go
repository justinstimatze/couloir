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

// TestMatchVerificationCommandBeyondGoAndJS pins coverage this project's own
// real transcripts (Go/JS-heavy) never exercised. Every pattern list here has
// only ever been checked against one person's usage; this is the concrete,
// synthetic stand-in for a second developer's stack until a real one shows up.
func TestMatchVerificationCommandBeyondGoAndJS(t *testing.T) {
	cases := []struct {
		command string
		want    string
	}{
		{"bundle exec rspec spec/", "test"},
		{"rake test", "test"},
		{"./vendor/bin/phpunit --testdox", "test"},
		{"dotnet test MyProject.sln", "test"},
		{"mvn test", "test"},
		{"./gradlew test", "test"},
		{"mix test --cover", "test"},
		{"swift test", "test"},
		{"rubocop -a", "lint"},
		{"vendor/bin/phpstan analyse", "lint"},
		{"mix credo --strict", "lint"},
		{"swiftlint lint", "lint"},
		{"dotnet build -c Release", "build"},
		{"./gradlew build", "build"},
		{"swift build -c release", "build"},
	}
	for _, c := range cases {
		if got := matchVerificationCommand(c.command); got != c.want {
			t.Errorf("matchVerificationCommand(%q) = %q, want %q", c.command, got, c.want)
		}
	}
}

// TestMatchVerificationCommandHonestGaps documents patterns that are
// deliberately still unrecognized -- a project-local wrapper script has no
// stable, safely-matchable substring (a bare "test"/"check" token would
// collide with unrelated filenames and words), so it stays a real, named
// limitation rather than something the broadened list above quietly papers
// over.
func TestMatchVerificationCommandHonestGaps(t *testing.T) {
	cases := []string{
		"./scripts/verify.sh",
		"just check",
		"bazel test //...",
		"./run_checks.sh",
	}
	for _, c := range cases {
		if got := matchVerificationCommand(c); got != "" {
			t.Errorf("matchVerificationCommand(%q) = %q, want \"\" (documented gap, not a regression)", c, got)
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
