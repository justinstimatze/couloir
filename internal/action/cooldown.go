// Package action is the cooldown-gated nudge selector — at most one
// suggestion surfaced per turn, never more than once per category per
// cooldown window, never forced content when nothing qualifies.
package action

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// CooldownState is a single, global (not per-session) record of when
// each category's nudge was last shown -- the whole point is not
// repeating the same nudge too often regardless of which session shows
// it.
type CooldownState struct {
	LastShown map[string]time.Time `json:"last_shown"`
}

// CooldownPath is the default cooldown-state location, alongside the
// substrate.
func CooldownPath() (string, error) {
	dir, err := substrate.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "action-cooldown.json"), nil
}

// LoadCooldown reads the cooldown state. Missing or unreadable is a
// zero-value state (nothing shown yet), never an error. Decodes
// timestamps as raw strings first and parses each one individually, so
// a single unparseable entry is dropped rather than costing every
// other category its real record -- this file is shared across every
// concurrent session on the host (CooldownState's own doc comment), and
// decoding straight into map[string]time.Time means one bad value
// aborts the whole map: confirmed live 2026-08-30, a malformed
// hand-edited timestamp for one category silently wiped the other
// three for every session reading this file, not just the one that
// wrote the bad entry.
func LoadCooldown(path string) CooldownState {
	empty := CooldownState{LastShown: map[string]time.Time{}}
	b, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var raw struct {
		LastShown map[string]string `json:"last_shown"`
	}
	if json.Unmarshal(b, &raw) != nil {
		return empty
	}
	s := CooldownState{LastShown: map[string]time.Time{}}
	for category, ts := range raw.LastShown {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}
		s.LastShown[category] = t
	}
	return s
}

const (
	// cooldownLockStaleAfter bounds how old an unreleased lock file can
	// be before a waiter treats it as abandoned by a crashed holder
	// rather than actively held -- a hook call finishes in
	// milliseconds, so anything genuinely still holding the lock this
	// long is gone, not slow.
	cooldownLockStaleAfter = 5 * time.Second
	// cooldownLockWaitBudget bounds total time spent waiting for the
	// lock before giving up -- a hook must return promptly, and losing
	// one rare nudge to contention is the same trade this project
	// already makes everywhere else for a missed or delayed signal.
	cooldownLockWaitBudget = 250 * time.Millisecond
)

// CooldownLockPath is the lock guarding LoadCooldown+SaveCooldown's
// read-modify-write sequence. Unlike transcriptscan's lock, which is
// per-session, this one has to be global: the cooldown file itself is
// shared across every concurrent session on the host, so two sessions
// racing that sequence can each read the same old state and each write
// their own version back, and whichever writes second silently drops
// the first one's update -- confirmed live 2026-08-30, surfaced while
// pressure-testing this same file for an unrelated parsing bug.
func CooldownLockPath() (string, error) {
	dir, err := substrate.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "action-cooldown.lock"), nil
}

// AcquireCooldownLock blocks briefly for exclusive access, taking over
// a lock file older than cooldownLockStaleAfter rather than waiting it
// out. Returns ok=false if the wait budget runs out first; the caller
// should skip this turn's nudge rather than risk clobbering a
// concurrent session's update.
func AcquireCooldownLock(path string) (release func(), ok bool) {
	deadline := time.Now().Add(cooldownLockWaitBudget)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { _ = os.Remove(path) }, true
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > cooldownLockStaleAfter {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// SaveCooldown writes the cooldown state atomically.
func SaveCooldown(path string, s CooldownState) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".cooldown-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
