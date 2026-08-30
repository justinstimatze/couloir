package transcriptscan

import (
	"path/filepath"
	"testing"
)

func TestLoadCheckpointMissingFileIsZeroValue(t *testing.T) {
	c := LoadCheckpoint(filepath.Join(t.TempDir(), "nope.json"))
	if c.ByteOffset != 0 || c.LastModel != "" {
		t.Errorf("LoadCheckpoint on missing file = %+v, want zero value", c)
	}
}

func TestSaveLoadCheckpointRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checkpoint.json")
	want := Checkpoint{ByteOffset: 4096, LastModel: "claude-sonnet-5"}
	if err := SaveCheckpoint(path, want); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	got := LoadCheckpoint(path)
	if got != want {
		t.Errorf("LoadCheckpoint = %+v, want %+v", got, want)
	}
}
