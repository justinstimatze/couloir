package transcriptscan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptPathSlugsCwd(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got, err := TranscriptPath("/home/user/Documents/couloir", "abc-123")
	if err != nil {
		t.Fatalf("TranscriptPath: %v", err)
	}
	want := filepath.Join(home, ".claude", "projects", "-home-user-Documents-couloir", "abc-123.jsonl")
	if got != want {
		t.Errorf("TranscriptPath() = %q, want %q", got, want)
	}
}
