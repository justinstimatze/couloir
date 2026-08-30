package lens

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

func TestIsWorktreeAdd(t *testing.T) {
	cases := []struct {
		command string
		want    bool
	}{
		{"git worktree add ../foo feature-x", true},
		{"GIT WORKTREE ADD ../foo feature-x", true},
		{"git worktree list", false},
		{"git status", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isWorktreeAdd(c.command); got != c.want {
			t.Errorf("isWorktreeAdd(%q) = %v, want %v", c.command, got, c.want)
		}
	}
}

// withCursorDir points substrate.StateDir (and so CursorDir) at a temp
// directory for the duration of fn, via XDG_STATE_HOME.
func withCursorDir(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir, err := CursorDir()
	if err != nil {
		t.Fatalf("CursorDir: %v", err)
	}
	return dir
}

func writeCursorFile(t *testing.T, dir, sessionID string, mtime time.Time) {
	t.Helper()
	path := filepath.Join(dir, sessionID+".json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write cursor file: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
}

func TestCheckConcurrentSessionsFindsRecentOtherSession(t *testing.T) {
	dir := withCursorDir(t)
	now := time.Now()
	writeCursorFile(t, dir, "other-session", now.Add(-1*time.Minute))

	got, cur := CheckConcurrentSessions("this-session", Cursor{}, now, sequentialIDs())
	if got == nil {
		t.Fatal("expected a concurrent_sessions_observed observation")
	}
	if got.SignalType != substrate.SignalConcurrentSessionsObserved {
		t.Errorf("SignalType = %q, want %q", got.SignalType, substrate.SignalConcurrentSessionsObserved)
	}
	if got.Detail["concurrent_session_count"] != 1 {
		t.Errorf("Detail = %+v, want concurrent_session_count=1", got.Detail)
	}
	if cur.ConcurrencyCheckedAt == "" {
		t.Error("expected ConcurrencyCheckedAt to be set")
	}
}

func TestCheckConcurrentSessionsIgnoresOwnFile(t *testing.T) {
	dir := withCursorDir(t)
	now := time.Now()
	writeCursorFile(t, dir, "this-session", now)

	got, _ := CheckConcurrentSessions("this-session", Cursor{}, now, sequentialIDs())
	if got != nil {
		t.Errorf("got %+v, want nil (only the caller's own cursor file exists)", got)
	}
}

func TestCheckConcurrentSessionsIgnoresStaleFile(t *testing.T) {
	dir := withCursorDir(t)
	now := time.Now()
	writeCursorFile(t, dir, "other-session", now.Add(-1*time.Hour))

	got, _ := CheckConcurrentSessions("this-session", Cursor{}, now, sequentialIDs())
	if got != nil {
		t.Errorf("got %+v, want nil (other session's file is outside the concurrency window)", got)
	}
}

func TestCheckConcurrentSessionsThrottled(t *testing.T) {
	dir := withCursorDir(t)
	now := time.Now()
	writeCursorFile(t, dir, "other-session", now)
	cur := Cursor{ConcurrencyCheckedAt: now.Add(-1 * time.Minute).UTC().Format(time.RFC3339)}

	got, gotCur := CheckConcurrentSessions("this-session", cur, now, sequentialIDs())
	if got != nil {
		t.Errorf("got %+v, want nil (throttled — checked less than ConcurrencyCheckInterval ago)", got)
	}
	if gotCur.ConcurrencyCheckedAt != cur.ConcurrencyCheckedAt {
		t.Error("a throttled check must not advance ConcurrencyCheckedAt")
	}
}
