// Package transcriptscan is the second Lens mechanism: triggered on
// Stop/SessionStart (not live, unlike internal/lens's PreToolUse half),
// it reads the persisted session transcript for the three categories
// tool-call shape can't carry: context_mgmt, prompting_structure,
// model_routing. Same discipline as internal/lens: raw facts only, never
// a rung guess.
package transcriptscan

import (
	"os"
	"path/filepath"
	"strings"
)

// TranscriptPath resolves a session's transcript file, given the cwd
// Claude Code reports on the hook payload. Claude Code slugs a project's
// absolute path by replacing every "/" with "-" (confirmed against this
// project's own transcript file, e.g. "/home/user/Documents/couloir" ->
// "-home-user-Documents-couloir").
func TranscriptPath(cwd, sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	slug := strings.ReplaceAll(cwd, "/", "-")
	return filepath.Join(home, ".claude", "projects", slug, sessionID+".jsonl"), nil
}
