package substrate

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendWritesOneLinePerCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "observations.jsonl")

	obs1 := Observation{SchemaVersion: SchemaVersion, ID: "1", SessionID: "s1", Lens: LensPreToolUse, CategoryID: CategoryTrustQA, SignalType: SignalEditRecorded, ToolName: "Edit"}
	obs2 := Observation{SchemaVersion: SchemaVersion, ID: "2", SessionID: "s1", Lens: LensPreToolUse, CategoryID: CategoryTrustQA, SignalType: SignalInspectionAfterEdit, ToolName: "Read"}

	if err := Append(path, obs1); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := Append(path, obs2); err != nil {
		t.Fatalf("second append: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	var got Observation
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if got.ID != "1" || got.SignalType != SignalEditRecorded {
		t.Errorf("line 1 = %+v, want id=1 signal=%s", got, SignalEditRecorded)
	}

	if err := json.Unmarshal([]byte(lines[1]), &got); err != nil {
		t.Fatalf("unmarshal line 2: %v", err)
	}
	if got.ID != "2" || got.SignalType != SignalInspectionAfterEdit {
		t.Errorf("line 2 = %+v, want id=2 signal=%s", got, SignalInspectionAfterEdit)
	}
}

func TestTailReturnsLastNInOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "observations.jsonl")
	for i := 1; i <= 5; i++ {
		id := string(rune('0' + i))
		if err := Append(path, Observation{ID: id, SignalType: SignalEditRecorded}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, err := Tail(path, 3)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d observations, want 3", len(got))
	}
	want := []string{"3", "4", "5"}
	for i, o := range got {
		if o.ID != want[i] {
			t.Errorf("got[%d].ID = %q, want %q", i, o.ID, want[i])
		}
	}
}

func TestTailFewerThanNReturnsAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "observations.jsonl")
	if err := Append(path, Observation{ID: "1", SignalType: SignalEditRecorded}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d observations, want 1", len(got))
	}
}

func TestTailMissingFileReturnsEmpty(t *testing.T) {
	got, err := Tail(filepath.Join(t.TempDir(), "nope.jsonl"), 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d observations, want 0", len(got))
	}
}

func TestStateDirRespectsXDGStateHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	dir, err := StateDir()
	if err != nil {
		t.Fatalf("StateDir: %v", err)
	}
	want := filepath.Join(tmp, "couloir")
	if dir != want {
		t.Errorf("StateDir() = %q, want %q", dir, want)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("StateDir() did not create %q", dir)
	}
}
