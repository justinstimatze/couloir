package calibration

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/gate"
)

func seqIDs() func() string {
	n := 0
	return func() string {
		n++
		return string(rune('a' - 1 + n))
	}
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		n++
	}
	return n
}

func TestRecordPredictionsLogsFirstFloorEstimate(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calibration.jsonl")
	statePath := filepath.Join(dir, "last-logged.json")

	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 4, Confidence: "high", AsOf: time.Now()},
	}
	RecordPredictions(logPath, statePath, estimates, time.Now(), seqIDs())

	if got := countLines(t, logPath); got != 1 {
		t.Fatalf("log has %d lines, want 1", got)
	}
}

func TestRecordPredictionsSkipsNonFloorStates(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calibration.jsonl")
	statePath := filepath.Join(dir, "last-logged.json")

	estimates := []gate.RungEstimate{
		{CategoryID: "model_routing", State: gate.StateBanded, RungMin: 2, RungMax: 4, Confidence: "low", AsOf: time.Now()},
		{CategoryID: "trust_qa", State: gate.StateInsufficientSignal, AsOf: time.Now()},
		{CategoryID: "agent_invocation", State: gate.StateUnmapped, UnmappedValues: []string{"auto"}, AsOf: time.Now()},
	}
	RecordPredictions(logPath, statePath, estimates, time.Now(), seqIDs())

	if got := countLines(t, logPath); got != 0 {
		t.Fatalf("log has %d lines, want 0 (no floor estimates)", got)
	}
}

func TestRecordPredictionsDedupsUnchangedRung(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calibration.jsonl")
	statePath := filepath.Join(dir, "last-logged.json")

	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 4, Confidence: "high", AsOf: time.Now()},
	}
	RecordPredictions(logPath, statePath, estimates, time.Now(), seqIDs())
	RecordPredictions(logPath, statePath, estimates, time.Now(), seqIDs())
	RecordPredictions(logPath, statePath, estimates, time.Now(), seqIDs())

	if got := countLines(t, logPath); got != 1 {
		t.Fatalf("log has %d lines, want 1 (unchanged rung must not re-log)", got)
	}
}

func TestRecordPredictionsLogsAgainOnRungChange(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calibration.jsonl")
	statePath := filepath.Join(dir, "last-logged.json")

	RecordPredictions(logPath, statePath,
		[]gate.RungEstimate{{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 4, Confidence: "high", AsOf: time.Now()}},
		time.Now(), seqIDs())
	RecordPredictions(logPath, statePath,
		[]gate.RungEstimate{{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 5, Confidence: "high", AsOf: time.Now()}},
		time.Now(), seqIDs())

	if got := countLines(t, logPath); got != 2 {
		t.Fatalf("log has %d lines, want 2 (rung 4 -> 5 is a real change)", got)
	}
}
