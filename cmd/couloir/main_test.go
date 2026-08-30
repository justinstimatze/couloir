package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/justinstimatze/couloir/internal/reasoner"
)

func TestValidSessionID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", true},
		{"test-1", true},
		{"", false},
		{"has a space", false},
		{"has/a/slash", false},
	}
	for _, c := range cases {
		if got := validSessionID(c.id); got != c.want {
			t.Errorf("validSessionID(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

func TestGenIDIsUniqueAndNonEmpty(t *testing.T) {
	a, b := genID(), genID()
	if a == "" || b == "" {
		t.Fatal("genID returned an empty string")
	}
	if a == b {
		t.Errorf("genID returned the same value twice: %q", a)
	}
}

func TestBuildVersionFallsBackToDev(t *testing.T) {
	// version defaults to "dev" in a test binary with no ldflags override
	// and no VCS build info attached.
	if got := buildVersion(); got == "" {
		t.Error("buildVersion() returned an empty string")
	}
}

// withStdin redirects os.Stdin to content for the duration of fn, and
// restores it after. The observe/transcript-scan/nudge dispatch
// branches all read stdin directly, so this is how their fail-open
// behavior gets exercised without a subprocess.
func withStdin(t *testing.T, content string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err := w.WriteString(content); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()
	fn()
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return buf.String()
}

func TestRunTranscriptScanMalformedInputDoesNotPanic(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withStdin(t, "not json at all", runTranscriptScan) // must not panic
}

func TestRunTranscriptScanMissingSessionIDIsNoop(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withStdin(t, `{"cwd":"/tmp"}`, runTranscriptScan) // no session_id -> must not panic, nothing written
}

func TestRunNudgePrintsNothingWithNoObservations(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var out string
	withStdin(t, `{}`, func() {
		out = captureStdout(t, runNudge)
	})
	if out != "" {
		t.Errorf("runNudge with an empty substrate printed %q, want nothing (insufficient signal everywhere)", out)
	}
}

func TestRunNudgeMalformedInputDoesNotPanic(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withStdin(t, "not json at all", runNudge) // must not panic
}

func TestRenderNudgeOutputCautionOnlyDoesNotClaimRungZero(t *testing.T) {
	s := &reasoner.Suggestion{
		CategoryID: "parallelization",
		Confidence: "medium",
		Caution: &reasoner.Caution{
			Definition: "parallel multi-agent sessions can burn through a plan's usage limits fast",
			SourceName: "ecliptik (HN handle)",
			SourceURL:  "https://news.ycombinator.com/item?id=47221592",
		},
	}
	out, err := renderNudgeOutput(s)
	if err != nil {
		t.Fatalf("renderNudgeOutput: %v", err)
	}
	if strings.Contains(out, "rung 0") {
		t.Errorf("output = %q, must not claim rung 0 for a caution-only suggestion", out)
	}
	if !strings.Contains(out, "ecliptik") || !strings.Contains(out, "usage limits") {
		t.Errorf("output = %q, want the caution text and source cited", out)
	}
}

func TestRenderNudgeOutputTellsTheAssistantToRelayIt(t *testing.T) {
	// A ripe, cooldown-cleared suggestion has already had its one
	// judgment call made server-side -- nothing should be left for the
	// assistant to decide except how to phrase it, so the injected
	// context must say so explicitly rather than trusting the assistant
	// to notice and relay it unprompted.
	s := &reasoner.Suggestion{
		CategoryID:  "trust_qa",
		CurrentRung: 3,
		Confidence:  "high",
	}
	out, err := renderNudgeOutput(s)
	if err != nil {
		t.Fatalf("renderNudgeOutput: %v", err)
	}
	if !strings.Contains(out, "Relay it to the user") {
		t.Errorf("output = %q, want an explicit relay instruction for the assistant", out)
	}
}
