package lens

import "strings"

// verificationPatterns match a Bash command against a curated test/lint/
// build/typecheck category. Only the matched label is ever stored — never
// the raw command string, which can carry secrets or reveal project
// layout even in a local, single-user state dir.
var verificationPatterns = []struct {
	label    string
	matchers []string
}{
	{"test", []string{
		"go test", "npm test", "npm run test", "pytest", "yarn test", "cargo test", "make test", "jest", "vitest",
		// Only ever verified against this project's own real transcripts,
		// which are Go/JS-heavy -- these ecosystem-specific commands
		// broaden coverage without narrow-matching against anything
		// actually seen, so they're checked against test_data.csv-style
		// false-positive risk (a bare "test"/"build" token would collide
		// with ordinary filenames/words) rather than real usage.
		"rspec", "rake test", "phpunit", "dotnet test", "mvn test", "gradle test", "gradlew test", "mix test", "swift test",
	}},
	{"lint", []string{
		"golangci-lint", "eslint", "ruff", "pylint", "cargo clippy", "make lint", "go vet",
		"rubocop", "phpstan", "psalm", "mix credo", "swiftlint", "checkstyle", "ktlint",
	}},
	{"typecheck", []string{"tsc", "mypy", "go build"}},
	{"build", []string{
		"make build", "npm run build", "cargo build", "docker build",
		"dotnet build", "mvn package", "gradle build", "gradlew build", "swift build",
	}},
}

// matchVerificationCommand reports the category label for a Bash command,
// or "" if it doesn't match any curated verification pattern.
func matchVerificationCommand(command string) string {
	cmd := strings.ToLower(command)
	for _, p := range verificationPatterns {
		for _, m := range p.matchers {
			if strings.Contains(cmd, m) {
				return p.label
			}
		}
	}
	return ""
}

// browserOrE2EPatterns are tool_name prefixes that indicate the agent
// verified its own work against a real interface, not just static
// analysis.
var browserOrE2EPatterns = []string{
	"mcp__claude-in-chrome__",
	"mcp__playwright__",
	"mcp__puppeteer__",
}

func isBrowserOrE2ETool(toolName string) bool {
	for _, p := range browserOrE2EPatterns {
		if strings.HasPrefix(toolName, p) {
			return true
		}
	}
	return false
}
