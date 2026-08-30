package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyAddsHookToEmptySettings(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	changes := s.Apply("/home/user/go/bin/couloir")
	if len(changes) != len(Hooks) {
		t.Fatalf("changes = %+v, want %d added (one per registered hook)", changes, len(Hooks))
	}
	for _, c := range changes {
		if c.Action != "added" {
			t.Errorf("change for %s/%s = %q, want added", c.Hook.Event, c.Hook.Sub, c.Action)
		}
	}

	reg := s.Registered()
	if reg["PreToolUse"] != "/home/user/go/bin/couloir observe" {
		t.Errorf("Registered()[PreToolUse] = %q, want the couloir observe command", reg["PreToolUse"])
	}
	if reg["Stop"] != "/home/user/go/bin/couloir transcript-scan" {
		t.Errorf("Registered()[Stop] = %q, want the couloir transcript-scan command", reg["Stop"])
	}
	if reg["SessionStart"] != "/home/user/go/bin/couloir transcript-scan" {
		t.Errorf("Registered()[SessionStart] = %q, want the couloir transcript-scan command", reg["SessionStart"])
	}
	if reg["UserPromptSubmit"] != "/home/user/go/bin/couloir nudge" {
		t.Errorf("Registered()[UserPromptSubmit] = %q, want the couloir nudge command", reg["UserPromptSubmit"])
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	s, _ := Load(filepath.Join(t.TempDir(), "missing.json"))
	s.Apply("/home/user/go/bin/couloir")
	changes := s.Apply("/home/user/go/bin/couloir")
	if len(changes) != len(Hooks) {
		t.Fatalf("second Apply changes = %+v, want %d unchanged", changes, len(Hooks))
	}
	for _, c := range changes {
		if c.Action != "unchanged" {
			t.Errorf("change for %s/%s = %q, want unchanged", c.Hook.Event, c.Hook.Sub, c.Action)
		}
	}
}

func TestApplyRetargetsExistingRegistration(t *testing.T) {
	s, _ := Load(filepath.Join(t.TempDir(), "missing.json"))
	s.Apply("/old/path/couloir")
	changes := s.Apply("/new/path/couloir")
	if len(changes) != len(Hooks) {
		t.Fatalf("changes = %+v, want %d updated from the old path", changes, len(Hooks))
	}
	for _, c := range changes {
		if c.Action != "updated" || !strings.HasPrefix(c.Was, "/old/path/couloir ") {
			t.Errorf("change for %s/%s = %+v, want updated from an /old/path/couloir command", c.Hook.Event, c.Hook.Sub, c)
		}
	}
	if s.Registered()["PreToolUse"] != "/new/path/couloir observe" {
		t.Errorf("Registered()[PreToolUse] = %q, want the new path", s.Registered()["PreToolUse"])
	}
	if s.Registered()["UserPromptSubmit"] != "/new/path/couloir nudge" {
		t.Errorf("Registered()[UserPromptSubmit] = %q, want the new path", s.Registered()["UserPromptSubmit"])
	}
}

func TestSavePreservesUnrelatedKeysAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	original := `{"model":"opus","hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo hi"}]}]}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("seed settings.json: %v", err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	s.Apply("/home/user/go/bin/couloir")
	backup, err := s.Save(path)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if backup == "" {
		t.Fatal("expected a backup path")
	}
	backupContent, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backupContent) != original {
		t.Errorf("backup content = %q, want the original file", backupContent)
	}

	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved settings: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(saved, &m); err != nil {
		t.Fatalf("unmarshal saved settings: %v", err)
	}
	if m["model"] != "opus" {
		t.Errorf("model = %v, want opus (unrelated key must survive)", m["model"])
	}
	hooks, _ := m["hooks"].(map[string]any)
	stopGroups, _ := hooks["Stop"].([]any)
	if len(stopGroups) != 2 {
		t.Fatalf("hooks.Stop has %d groups, want 2 (the pre-existing one plus couloir's own)", len(stopGroups))
	}
	foundEchoHi, foundCouloir := false, false
	for _, g := range stopGroups {
		gm, _ := g.(map[string]any)
		inner, _ := gm["hooks"].([]any)
		for _, e := range inner {
			em, _ := e.(map[string]any)
			cmd, _ := em["command"].(string)
			switch cmd {
			case "echo hi":
				foundEchoHi = true
			case "/home/user/go/bin/couloir transcript-scan":
				foundCouloir = true
			}
		}
	}
	if !foundEchoHi {
		t.Error("expected the pre-existing Stop hook (echo hi) to survive")
	}
	if !foundCouloir {
		t.Error("expected couloir's own transcript-scan Stop hook to be registered alongside it")
	}
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("expected a PreToolUse hook to be registered")
	}
}

func TestRemoveStripsOnlyCouloirEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	s, _ := Load(path)
	s.Apply("/home/user/go/bin/couloir")
	if _, err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	s2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	changes := s2.Remove()
	if len(changes) != len(Hooks) {
		t.Fatalf("changes = %+v, want %d removed", changes, len(Hooks))
	}
	for _, c := range changes {
		if c.Action != "removed" {
			t.Errorf("change for %s/%s = %q, want removed", c.Hook.Event, c.Hook.Sub, c.Action)
		}
	}
	if len(s2.Registered()) != 0 {
		t.Errorf("Registered() = %+v, want empty after Remove", s2.Registered())
	}
}
