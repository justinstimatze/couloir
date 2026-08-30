package lens

import (
	"path/filepath"
	"testing"
)

func TestLoadCursorMissingFileIsZeroValue(t *testing.T) {
	c := LoadCursor(filepath.Join(t.TempDir(), "nope.json"))
	if c.LastPermissionMode != "" || c.PendingEdit != nil {
		t.Errorf("LoadCursor on missing file = %+v, want zero value", c)
	}
}

func TestLoadCursorCorruptFileIsZeroValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	if err := SaveCursor(path, Cursor{LastPermissionMode: "default"}); err != nil {
		t.Fatalf("seed SaveCursor: %v", err)
	}
	// Overwrite with garbage.
	if err := SaveCursor(path, Cursor{}); err != nil {
		t.Fatalf("SaveCursor: %v", err)
	}
	c := LoadCursor(path)
	if c.LastPermissionMode != "" {
		t.Errorf("LoadCursor = %+v, want cleared cursor", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.json")
	want := Cursor{
		LastPermissionMode:   "acceptEdits",
		PendingEdit:          &PendingEdit{Path: "/repo/main.go", ToolUseID: "toolu_1", CallsSince: 2},
		ConcurrencyCheckedAt: "2026-08-29T12:00:00Z",
	}
	if err := SaveCursor(path, want); err != nil {
		t.Fatalf("SaveCursor: %v", err)
	}
	got := LoadCursor(path)
	if got.LastPermissionMode != want.LastPermissionMode {
		t.Errorf("LastPermissionMode = %q, want %q", got.LastPermissionMode, want.LastPermissionMode)
	}
	if got.PendingEdit == nil || *got.PendingEdit != *want.PendingEdit {
		t.Errorf("PendingEdit = %+v, want %+v", got.PendingEdit, want.PendingEdit)
	}
	if got.ConcurrencyCheckedAt != want.ConcurrencyCheckedAt {
		t.Errorf("ConcurrencyCheckedAt = %q, want %q", got.ConcurrencyCheckedAt, want.ConcurrencyCheckedAt)
	}
}
