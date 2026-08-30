package lens

import (
	"os"
	"strings"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// worktreeAddMatchers, same style as verify_patterns.go's
// matchVerificationCommand: a lowercased substring match, no regexp.
var worktreeAddMatchers = []string{"git worktree add"}

// isWorktreeAdd reports whether a Bash command adopts git's worktree
// isolation — the mechanical precondition data/ladder.json's
// parallelization rung 2 (m11) names for real concurrent session use.
func isWorktreeAdd(command string) bool {
	cmd := strings.ToLower(command)
	for _, m := range worktreeAddMatchers {
		if strings.Contains(cmd, m) {
			return true
		}
	}
	return false
}

// ConcurrencyCheckInterval throttles CheckConcurrentSessions so a
// PreToolUse call (which can fire many times a minute) doesn't sweep the
// cursor directory on every single one.
const ConcurrencyCheckInterval = 5 * time.Minute

// ConcurrencyWindow is how recently another session's cursor file must
// have been touched to count as "currently active." A starting default,
// not a measured value — cursor files are written on every PreToolUse
// call, so this is a proxy for recent activity, not a live process
// check: a session that stopped inside this window still looks active.
const ConcurrencyWindow = 5 * time.Minute

// CheckConcurrentSessions scans the cursor directory for other sessions
// with recent activity, throttled by cur.ConcurrencyCheckedAt. Returns a
// nil observation when throttled, when the scan can't run, or when no
// concurrent activity is found — the caller should still persist the
// returned cursor, since ConcurrencyCheckedAt advances even on a
// zero-result check.
func CheckConcurrentSessions(sessionID string, cur Cursor, now time.Time, newID func() string) (*substrate.Observation, Cursor) {
	if cur.ConcurrencyCheckedAt != "" {
		if last, err := time.Parse(time.RFC3339, cur.ConcurrencyCheckedAt); err == nil && now.Sub(last) < ConcurrencyCheckInterval {
			return nil, cur
		}
	}
	cur.ConcurrencyCheckedAt = now.UTC().Format(time.RFC3339)

	dir, err := CursorDir()
	if err != nil {
		return nil, cur
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, cur
	}

	count := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if strings.TrimSuffix(name, ".json") == sessionID {
			continue
		}
		info, err := e.Info()
		if err != nil || now.Sub(info.ModTime()) > ConcurrencyWindow {
			continue
		}
		count++
	}
	if count == 0 {
		return nil, cur
	}

	return &substrate.Observation{
		SchemaVersion: substrate.SchemaVersion,
		ID:            newID(),
		ObservedAt:    now.UTC().Format(time.RFC3339),
		SessionID:     sessionID,
		Lens:          substrate.LensPreToolUse,
		CategoryID:    substrate.CategoryParallelization,
		SignalType:    substrate.SignalConcurrentSessionsObserved,
		Detail:        map[string]any{"concurrent_session_count": count},
	}, cur
}
