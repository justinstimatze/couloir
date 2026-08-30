package calibration

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendObservationGrounding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calibration.jsonl")

	e := Entry{
		SchemaVersion:        SchemaVersion,
		ID:                   "c1",
		LoggedAt:             "2026-08-29T22:10:00Z",
		Kind:                 KindObservationGrounding,
		SubjectObservationID: "obs1",
		CategoryID:           "trust_qa",
		Predict:              nil,
		Verdict: &Verdict{
			Correct:   true,
			CheckedBy: "human",
			CheckedAt: "2026-08-29T22:12:00Z",
		},
	}
	if err := Append(path, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("expected one line, got none")
	}

	var got Entry
	if err := json.Unmarshal(sc.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Kind != KindObservationGrounding {
		t.Errorf("Kind = %q, want %q", got.Kind, KindObservationGrounding)
	}
	if got.Predict != nil {
		t.Errorf("Predict = %+v, want nil (Gate does not exist yet)", got.Predict)
	}
	if got.Verdict == nil || !got.Verdict.Correct {
		t.Errorf("Verdict = %+v, want Correct=true", got.Verdict)
	}
}
