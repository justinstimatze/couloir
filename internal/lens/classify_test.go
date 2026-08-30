package lens

import (
	"encoding/json"
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

func mustRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func findSignal(obs []substrate.Observation, signal string) *substrate.Observation {
	for i := range obs {
		if obs[i].SignalType == signal {
			return &obs[i]
		}
	}
	return nil
}

func TestClassifyPermissionModeSnapshotOnFirstSighting(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", PermissionMode: "default", ToolName: "Bash",
		ToolInput: mustRaw(t, map[string]string{"command": "ls"})}
	obs, cur := Classify(in, Cursor{}, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalPermissionModeSnapshot)
	if got == nil {
		t.Fatal("expected a permission_mode_snapshot observation")
	}
	if got.CategoryID != substrate.CategoryAgentInvocation {
		t.Errorf("category = %q, want %q", got.CategoryID, substrate.CategoryAgentInvocation)
	}
	if _, ok := got.Detail["previous_permission_mode"]; ok {
		t.Errorf("first sighting should carry no previous_permission_mode, got %+v", got.Detail)
	}
	if cur.LastPermissionMode != "default" {
		t.Errorf("cursor.LastPermissionMode = %q, want default", cur.LastPermissionMode)
	}
}

func TestClassifyPermissionModeChangeCarriesPrevious(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", PermissionMode: "acceptEdits", ToolName: "Edit",
		ToolInput: mustRaw(t, map[string]string{"file_path": "/repo/main.go"})}
	obs, cur := Classify(in, Cursor{LastPermissionMode: "default"}, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalPermissionModeSnapshot)
	if got == nil {
		t.Fatal("expected a permission_mode_snapshot observation")
	}
	if got.Detail["previous_permission_mode"] != "default" {
		t.Errorf("Detail = %+v, want previous_permission_mode=default", got.Detail)
	}
	if cur.LastPermissionMode != "acceptEdits" {
		t.Errorf("cursor.LastPermissionMode = %q, want acceptEdits", cur.LastPermissionMode)
	}
}

func TestClassifyPlanModeSnapshotGoesToPromptingStructure(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", PermissionMode: "plan", ToolName: "Bash",
		ToolInput: mustRaw(t, map[string]string{"command": "ls"})}
	obs, _ := Classify(in, Cursor{LastPermissionMode: "default"}, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalPermissionModeSnapshot)
	if got == nil {
		t.Fatal("expected a permission_mode_snapshot observation")
	}
	if got.CategoryID != substrate.CategoryPromptingStructure {
		t.Errorf("category = %q, want %q (plan mode is a prompting_structure signal, not agent_invocation)",
			got.CategoryID, substrate.CategoryPromptingStructure)
	}
}

func TestClassifyNonPlanModeStillGoesToAgentInvocation(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", PermissionMode: "dontAsk", ToolName: "Bash",
		ToolInput: mustRaw(t, map[string]string{"command": "ls"})}
	obs, _ := Classify(in, Cursor{LastPermissionMode: "default"}, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalPermissionModeSnapshot)
	if got == nil {
		t.Fatal("expected a permission_mode_snapshot observation")
	}
	if got.CategoryID != substrate.CategoryAgentInvocation {
		t.Errorf("category = %q, want %q", got.CategoryID, substrate.CategoryAgentInvocation)
	}
}

func TestClassifyEditThenReadSamePathIsInspection(t *testing.T) {
	editIn := PreToolUseInput{SessionID: "s1", ToolName: "Edit", ToolUseID: "toolu_1",
		ToolInput: mustRaw(t, map[string]string{"file_path": "/repo/main.go"})}
	_, cur := Classify(editIn, Cursor{}, time.Now(), sequentialIDs())

	if cur.PendingEdit == nil || cur.PendingEdit.Path != "/repo/main.go" {
		t.Fatalf("expected a pending edit on /repo/main.go, got %+v", cur.PendingEdit)
	}

	readIn := PreToolUseInput{SessionID: "s1", ToolName: "Read",
		ToolInput: mustRaw(t, map[string]string{"file_path": "/repo/main.go"})}
	obs, cur2 := Classify(readIn, cur, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalInspectionAfterEdit)
	if got == nil {
		t.Fatal("expected an inspection_after_edit observation")
	}
	if got.Detail["edited_tool_use_id"] != "toolu_1" {
		t.Errorf("Detail = %+v, want edited_tool_use_id=toolu_1", got.Detail)
	}
	if cur2.PendingEdit != nil {
		t.Errorf("PendingEdit should be cleared after inspection, got %+v", cur2.PendingEdit)
	}
}

func TestClassifyEditThenVerificationCommandIsInspection(t *testing.T) {
	editIn := PreToolUseInput{SessionID: "s1", ToolName: "Write", ToolUseID: "toolu_1",
		ToolInput: mustRaw(t, map[string]string{"file_path": "/repo/main.go"})}
	_, cur := Classify(editIn, Cursor{}, time.Now(), sequentialIDs())

	bashIn := PreToolUseInput{SessionID: "s1", ToolName: "Bash",
		ToolInput: mustRaw(t, map[string]string{"command": "go test ./..."})}
	obs, cur2 := Classify(bashIn, cur, time.Now(), sequentialIDs())

	if findSignal(obs, substrate.SignalVerificationCommandRan) == nil {
		t.Error("expected a verification_command_ran observation")
	}
	inspection := findSignal(obs, substrate.SignalInspectionAfterEdit)
	if inspection == nil {
		t.Fatal("expected an inspection_after_edit observation")
	}
	if inspection.Detail["inspecting_tool"] != "Bash:test" {
		t.Errorf("Detail = %+v, want inspecting_tool=Bash:test", inspection.Detail)
	}
	if cur2.PendingEdit != nil {
		t.Error("PendingEdit should be cleared after a verification command")
	}
}

func TestClassifyEditDifferentFileBeforeInspectionIsUninspected(t *testing.T) {
	editA := PreToolUseInput{SessionID: "s1", ToolName: "Edit", ToolUseID: "toolu_1",
		ToolInput: mustRaw(t, map[string]string{"file_path": "/repo/a.go"})}
	_, cur := Classify(editA, Cursor{}, time.Now(), sequentialIDs())

	editB := PreToolUseInput{SessionID: "s1", ToolName: "Edit", ToolUseID: "toolu_2",
		ToolInput: mustRaw(t, map[string]string{"file_path": "/repo/b.go"})}
	obs, cur2 := Classify(editB, cur, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalEditUninspectedBeforeNext)
	if got == nil {
		t.Fatal("expected an edit_uninspected_before_next_edit observation")
	}
	if got.SubjectPath != "/repo/a.go" {
		t.Errorf("SubjectPath = %q, want /repo/a.go", got.SubjectPath)
	}
	if cur2.PendingEdit == nil || cur2.PendingEdit.Path != "/repo/b.go" {
		t.Errorf("expected pending edit to move to /repo/b.go, got %+v", cur2.PendingEdit)
	}
}

func TestClassifyAgentToolRecordsRawFactOnly(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", ToolName: "Agent",
		ToolInput: mustRaw(t, map[string]string{"subagent_type": "Explore", "model": "sonnet"})}
	obs, _ := Classify(in, Cursor{}, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalSubagentInvoked)
	if got == nil {
		t.Fatal("expected a subagent_invoked observation")
	}
	if got.Detail["subagent_type"] != "Explore" || got.Detail["model"] != "sonnet" {
		t.Errorf("Detail = %+v, want subagent_type=Explore model=sonnet", got.Detail)
	}
}

func TestClassifyGitWorktreeAddIsRecorded(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", ToolName: "Bash",
		ToolInput: mustRaw(t, map[string]string{"command": "git worktree add ../foo feature-x"})}
	obs, _ := Classify(in, Cursor{}, time.Now(), sequentialIDs())

	got := findSignal(obs, substrate.SignalGitWorktreeAdded)
	if got == nil {
		t.Fatal("expected a git_worktree_added observation")
	}
	if got.CategoryID != substrate.CategoryParallelization {
		t.Errorf("category = %q, want %q", got.CategoryID, substrate.CategoryParallelization)
	}
}

func TestClassifyUnrelatedBashCommandNoWorktreeSignal(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", ToolName: "Bash",
		ToolInput: mustRaw(t, map[string]string{"command": "git status"})}
	obs, _ := Classify(in, Cursor{}, time.Now(), sequentialIDs())

	if findSignal(obs, substrate.SignalGitWorktreeAdded) != nil {
		t.Error("expected no git_worktree_added observation for an unrelated git command")
	}
}

func TestClassifyMalformedToolInputDoesNotPanic(t *testing.T) {
	in := PreToolUseInput{SessionID: "s1", ToolName: "Edit", ToolInput: json.RawMessage(`not json`)}
	obs, cur := Classify(in, Cursor{}, time.Now(), sequentialIDs())
	if findSignal(obs, substrate.SignalEditRecorded) == nil {
		t.Error("expected edit_recorded even with malformed tool_input (empty path)")
	}
	if cur.PendingEdit == nil {
		t.Error("expected a pending edit even with an empty path")
	}
}
