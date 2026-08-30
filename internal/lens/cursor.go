package lens

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// PendingEdit is an Edit/Write not yet resolved by a same-path Read or a
// matched verification command.
type PendingEdit struct {
	Path       string `json:"path"`
	ToolUseID  string `json:"tool_use_id"`
	CallsSince int    `json:"calls_since"`
}

// Cursor is the per-session bookkeeping the Lens needs to detect a
// same-session sequence. It is disposable scratch state couloir alone
// writes and reads — Gate never sees it, only the substrate.
type Cursor struct {
	LastPermissionMode   string       `json:"last_permission_mode,omitempty"`
	PendingEdit          *PendingEdit `json:"pending_edit,omitempty"`
	ConcurrencyCheckedAt string       `json:"concurrency_checked_at,omitempty"`
}

// CursorDir is where per-session cursor files live.
func CursorDir() (string, error) {
	dir, err := substrate.StateDir()
	if err != nil {
		return "", err
	}
	cursorDir := filepath.Join(dir, "cursor")
	return cursorDir, os.MkdirAll(cursorDir, 0o755)
}

// CursorPath is one session's cursor file.
func CursorPath(sessionID string) (string, error) {
	dir, err := CursorDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionID+".json"), nil
}

// LoadCursor reads a session's cursor. A missing or corrupt file is a
// zero-value cursor, never an error.
func LoadCursor(path string) Cursor {
	b, err := os.ReadFile(path)
	if err != nil {
		return Cursor{}
	}
	var c Cursor
	if json.Unmarshal(b, &c) != nil {
		return Cursor{}
	}
	return c
}

// SaveCursor writes a session's cursor atomically (temp file + rename).
func SaveCursor(path string, c Cursor) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".cursor-*.tmp")
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
