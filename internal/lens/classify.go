package lens

import (
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// Classify turns one PreToolUse call into zero or more raw observations,
// threading the session cursor forward. It never errors: a hook that
// fails loudly in front of every tool call is worse than a lost
// observation. Every emitted fact is checked against "is this literally
// true," never "does this plausibly correlate with rung N" — that
// interpretive step is Gate's, not built yet.
func Classify(in PreToolUseInput, cur Cursor, now time.Time, newID func() string) ([]substrate.Observation, Cursor) {
	var obs []substrate.Observation
	ts := now.UTC().Format(time.RFC3339)

	emit := func(category, signal, subjectPath string, detail map[string]any) {
		obs = append(obs, substrate.Observation{
			SchemaVersion:  substrate.SchemaVersion,
			ID:             newID(),
			ObservedAt:     ts,
			SessionID:      in.SessionID,
			Lens:           substrate.LensPreToolUse,
			CategoryID:     category,
			SignalType:     signal,
			ToolName:       in.ToolName,
			ToolUseID:      in.ToolUseID,
			PermissionMode: in.PermissionMode,
			SubjectPath:    subjectPath,
			Detail:         detail,
		})
	}

	if in.PermissionMode != "" && in.PermissionMode != cur.LastPermissionMode {
		detail := map[string]any{}
		if cur.LastPermissionMode != "" {
			detail["previous_permission_mode"] = cur.LastPermissionMode
		}
		// "plan" mode is prompting_structure rung 5's signal (deliberate
		// plan-first workflow), not an agent_invocation supervision level
		// -- every other mode stays under agent_invocation, where the
		// rung table actually grounds it.
		category := substrate.CategoryAgentInvocation
		if in.PermissionMode == "plan" {
			category = substrate.CategoryPromptingStructure
		}
		emit(category, substrate.SignalPermissionModeSnapshot, "", detail)
		cur.LastPermissionMode = in.PermissionMode
	}

	if in.ToolName == "AskUserQuestion" {
		emit(substrate.CategoryAgentInvocation, substrate.SignalAskUserQuestionInvoked, "",
			map[string]any{"question_count": askUserQuestionCount(in.ToolInput)})
	}

	switch in.ToolName {
	case "Edit", "Write":
		path := editWritePath(in.ToolInput)
		if cur.PendingEdit != nil && cur.PendingEdit.Path != path {
			emit(substrate.CategoryTrustQA, substrate.SignalEditUninspectedBeforeNext, cur.PendingEdit.Path,
				map[string]any{
					"edited_tool_use_id": cur.PendingEdit.ToolUseID,
					"calls_since":        cur.PendingEdit.CallsSince,
				})
		}
		cur.PendingEdit = &PendingEdit{Path: path, ToolUseID: in.ToolUseID, CallsSince: 0}
		emit(substrate.CategoryTrustQA, substrate.SignalEditRecorded, path, nil)

	default:
		if cur.PendingEdit != nil {
			cur.PendingEdit.CallsSince++
			matched := false
			inspectingTool := ""

			switch {
			case in.ToolName == "Read" && readPath(in.ToolInput) == cur.PendingEdit.Path && cur.PendingEdit.Path != "":
				matched = true
				inspectingTool = "Read"
			case in.ToolName == "Bash":
				if label := matchVerificationCommand(bashCommand(in.ToolInput)); label != "" {
					emit(substrate.CategoryTrustQA, substrate.SignalVerificationCommandRan, "", map[string]any{"category": label})
					matched = true
					inspectingTool = "Bash:" + label
				}
			}

			if matched {
				emit(substrate.CategoryTrustQA, substrate.SignalInspectionAfterEdit, cur.PendingEdit.Path,
					map[string]any{
						"edited_tool_use_id": cur.PendingEdit.ToolUseID,
						"gap_tool_calls":     cur.PendingEdit.CallsSince,
						"inspecting_tool":    inspectingTool,
					})
				cur.PendingEdit = nil
			}
		}
	}

	if in.ToolName == "LSP" {
		emit(substrate.CategoryTrustQA, substrate.SignalLSPDiagnosticsUsed, "", nil)
	}

	if in.ToolName == "Agent" {
		subagentType, model := agentFields(in.ToolInput)
		emit(substrate.CategoryTrustQA, substrate.SignalSubagentInvoked, "",
			map[string]any{"subagent_type": subagentType, "model": model})
	}

	if isBrowserOrE2ETool(in.ToolName) {
		emit(substrate.CategoryTrustQA, substrate.SignalBrowserOrE2EToolInvoked, "", nil)
	}

	if in.ToolName == "Bash" && isWorktreeAdd(bashCommand(in.ToolInput)) {
		emit(substrate.CategoryParallelization, substrate.SignalGitWorktreeAdded, "", nil)
	}

	return obs, cur
}
