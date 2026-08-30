package transcriptscan

import (
	"strings"
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

func sequentialIDs() func() string {
	n := 0
	return func() string {
		n++
		return string(rune('a' - 1 + n))
	}
}

func findSignal(obs []substrate.Observation, signal string) *substrate.Observation {
	for i := range obs {
		if obs[i].SignalType == signal {
			return &obs[i]
		}
	}
	return nil
}

func countSignal(obs []substrate.Observation, signal string) int {
	n := 0
	for _, o := range obs {
		if o.SignalType == signal {
			n++
		}
	}
	return n
}

func TestClassifyCompactBoundaryManual(t *testing.T) {
	line := `{"type":"system","subtype":"compact_boundary","compactMetadata":{"trigger":"manual"}}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())

	got := findSignal(res.Observations, substrate.SignalCompactBoundaryObserved)
	if got == nil {
		t.Fatal("expected a compact_boundary_observed observation")
	}
	if got.CategoryID != substrate.CategoryContextMgmt {
		t.Errorf("category = %q, want %q", got.CategoryID, substrate.CategoryContextMgmt)
	}
	if got.Detail["trigger"] != "manual" {
		t.Errorf("Detail = %+v, want trigger=manual", got.Detail)
	}
}

func TestClassifyCompactBoundaryAutomatic(t *testing.T) {
	line := `{"type":"system","subtype":"compact_boundary","compactMetadata":{"trigger":"auto"}}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())

	got := findSignal(res.Observations, substrate.SignalCompactBoundaryObserved)
	if got == nil || got.Detail["trigger"] != "auto" {
		t.Fatalf("expected trigger=auto, got %+v", res.Observations)
	}
}

func TestClassifyIgnoresUnrelatedSystemSubtypes(t *testing.T) {
	line := `{"type":"system","subtype":"turn_duration"}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())
	if len(res.Observations) != 0 {
		t.Errorf("expected no observations for an unrelated system subtype, got %+v", res.Observations)
	}
}

func TestClassifyModelSwitch(t *testing.T) {
	lines := `{"type":"assistant","message":{"model":"claude-sonnet-5","usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"assistant","message":{"model":"claude-opus-5","usage":{"input_tokens":100,"output_tokens":50}}}
`
	res := Classify(strings.NewReader(lines), "", "s1", time.Now(), sequentialIDs())

	got := findSignal(res.Observations, substrate.SignalModelSwitchObserved)
	if got == nil {
		t.Fatal("expected a model_switch_observed observation")
	}
	if got.Detail["previous_model"] != "claude-sonnet-5" || got.Detail["new_model"] != "claude-opus-5" {
		t.Errorf("Detail = %+v, want previous=claude-sonnet-5 new=claude-opus-5", got.Detail)
	}
	if res.LastModel != "claude-opus-5" {
		t.Errorf("LastModel = %q, want claude-opus-5", res.LastModel)
	}
}

func TestClassifyNoModelSwitchWhenModelConstant(t *testing.T) {
	lines := `{"type":"assistant","message":{"model":"claude-sonnet-5","usage":{"input_tokens":10,"output_tokens":5}}}
{"type":"assistant","message":{"model":"claude-sonnet-5","usage":{"input_tokens":10,"output_tokens":5}}}
`
	res := Classify(strings.NewReader(lines), "", "s1", time.Now(), sequentialIDs())
	if findSignal(res.Observations, substrate.SignalModelSwitchObserved) != nil {
		t.Error("expected no model_switch_observed when the model never changes")
	}
}

func TestClassifyModelSwitchAcrossScansUsesPrevModel(t *testing.T) {
	// Simulates a second scan pass, seeded with the checkpoint's LastModel
	// from a prior pass, seeing a switch on the very first line.
	line := `{"type":"assistant","message":{"model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":5}}}` + "\n"
	res := Classify(strings.NewReader(line), "claude-sonnet-5", "s1", time.Now(), sequentialIDs())

	got := findSignal(res.Observations, substrate.SignalModelSwitchObserved)
	if got == nil {
		t.Fatal("expected a model_switch_observed observation using the seeded prevModel")
	}
}

func TestClassifyWindowUtilizationSample(t *testing.T) {
	line := `{"type":"assistant","message":{"model":"claude-sonnet-5","usage":{"input_tokens":1000,"cache_read_input_tokens":500,"output_tokens":100}}}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())

	got := findSignal(res.Observations, substrate.SignalWindowUtilizationSample)
	if got == nil {
		t.Fatal("expected a window_utilization_sample observation")
	}
	if got.Detail["total_tokens"] != 1600 {
		t.Errorf("total_tokens = %v, want 1600", got.Detail["total_tokens"])
	}
	if got.ModelID != "claude-sonnet-5" {
		t.Errorf("ModelID = %q, want claude-sonnet-5", got.ModelID)
	}
}

func TestClassifyTypedPromptWithURLAndPath(t *testing.T) {
	line := `{"type":"user","promptSource":"typed","message":{"content":"read /home/user/proj/main.go and see https://example.com/docs"}}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())

	if findSignal(res.Observations, substrate.SignalTypedPromptObserved) == nil {
		t.Error("expected a typed_prompt_observed observation")
	}
	if findSignal(res.Observations, substrate.SignalPromptContainsPath) == nil {
		t.Error("expected a prompt_contains_absolute_path observation")
	}
	if findSignal(res.Observations, substrate.SignalPromptContainsURL) == nil {
		t.Error("expected a prompt_contains_url observation")
	}
}

func TestClassifyPlainTypedPromptNoPathOrURL(t *testing.T) {
	line := `{"type":"user","promptSource":"typed","message":{"content":"fix the bug in the login flow"}}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())

	if findSignal(res.Observations, substrate.SignalTypedPromptObserved) == nil {
		t.Error("expected a typed_prompt_observed observation")
	}
	if findSignal(res.Observations, substrate.SignalPromptContainsPath) != nil {
		t.Error("did not expect a prompt_contains_absolute_path observation")
	}
	if findSignal(res.Observations, substrate.SignalPromptContainsURL) != nil {
		t.Error("did not expect a prompt_contains_url observation")
	}
}

func TestClassifySkipsNonTypedPromptSources(t *testing.T) {
	// promptSource "queued"/"system" covers hook feedback, skill
	// preambles, and slash-command expansions -- not real keystrokes,
	// and too coarse to distinguish from each other (see the plan's
	// rung-6 gap note), so none of them should produce a signal.
	lines := `{"type":"user","promptSource":"system","message":{"content":"Base directory for this skill: /home/user/.claude/skills/foo"}}
{"type":"user","promptSource":"queued","message":{"content":"Stop hook feedback: something happened"}}
`
	res := Classify(strings.NewReader(lines), "", "s1", time.Now(), sequentialIDs())
	if len(res.Observations) != 0 {
		t.Errorf("expected no observations from non-typed prompt sources, got %+v", res.Observations)
	}
}

func TestClassifySkipsArrayShapedUserContent(t *testing.T) {
	// A tool_result content block, not a real prompt.
	line := `{"type":"user","promptSource":"typed","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}}` + "\n"
	res := Classify(strings.NewReader(line), "", "s1", time.Now(), sequentialIDs())
	if len(res.Observations) != 0 {
		t.Errorf("expected no observations from array-shaped content, got %+v", res.Observations)
	}
}

func TestClassifyIgnoresUnrecognizedLineTypes(t *testing.T) {
	// This host's hook/plugin stack injects its own line types
	// (mode, permission-mode, agent-name, etc.) -- these must be
	// silently ignored, not treated as errors.
	lines := `{"type":"agent-name","name":"couloir"}
{"type":"mode","value":"default"}
` + `{"type":"assistant","message":{"model":"claude-sonnet-5","usage":{"input_tokens":10,"output_tokens":5}}}` + "\n"
	res := Classify(strings.NewReader(lines), "", "s1", time.Now(), sequentialIDs())
	if countSignal(res.Observations, substrate.SignalWindowUtilizationSample) != 1 {
		t.Errorf("expected exactly one window_utilization_sample despite unrecognized line types, got %+v", res.Observations)
	}
}

func TestClassifyMalformedLineDoesNotPanic(t *testing.T) {
	lines := "not json at all\n" + `{"type":"assistant","message":{"model":"claude-sonnet-5","usage":{"input_tokens":10,"output_tokens":5}}}` + "\n"
	res := Classify(strings.NewReader(lines), "", "s1", time.Now(), sequentialIDs())
	if findSignal(res.Observations, substrate.SignalWindowUtilizationSample) == nil {
		t.Error("expected the well-formed second line to still be classified after a malformed first line")
	}
}

func TestClassifyBytesReadTracksInput(t *testing.T) {
	lines := `{"type":"system","subtype":"turn_duration"}` + "\n" + `{"type":"system","subtype":"turn_duration"}` + "\n"
	res := Classify(strings.NewReader(lines), "", "s1", time.Now(), sequentialIDs())
	if res.BytesRead != int64(len(lines)) {
		t.Errorf("BytesRead = %d, want %d", res.BytesRead, len(lines))
	}
}
