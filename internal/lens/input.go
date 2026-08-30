// Package lens is the live PreToolUse observer: it reads one Claude Code
// tool call, extracts literal facts about it, and threads a small
// per-session cursor forward so it can detect same-session sequences
// (edit -> inspected-or-not) without reading Claude Code's own transcript,
// which is written asynchronously and can lag the current turn.
package lens

import (
	"encoding/json"
	"path/filepath"
)

// PreToolUseInput is the JSON payload Claude Code sends on stdin for a
// PreToolUse hook. tool_input's shape varies by tool_name, so it's kept
// raw and decoded per-tool by the helpers below.
type PreToolUseInput struct {
	SessionID      string          `json:"session_id"`
	CWD            string          `json:"cwd"`
	PermissionMode string          `json:"permission_mode"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	ToolUseID      string          `json:"tool_use_id"`
	ToolInput      json.RawMessage `json:"tool_input"`
}

// editWritePath pulls the target file out of an Edit/Write/Read call's
// tool_input, all of which carry the same file_path field.
func editWritePath(raw json.RawMessage) string {
	var v struct {
		FilePath string `json:"file_path"`
	}
	_ = json.Unmarshal(raw, &v)
	if v.FilePath == "" {
		return ""
	}
	return filepath.Clean(v.FilePath)
}

func readPath(raw json.RawMessage) string {
	return editWritePath(raw)
}

func bashCommand(raw json.RawMessage) string {
	var v struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &v)
	return v.Command
}

func askUserQuestionCount(raw json.RawMessage) int {
	var v struct {
		Questions []json.RawMessage `json:"questions"`
	}
	_ = json.Unmarshal(raw, &v)
	return len(v.Questions)
}

func agentFields(raw json.RawMessage) (subagentType, model string) {
	var v struct {
		SubagentType string `json:"subagent_type"`
		Model        string `json:"model"`
	}
	_ = json.Unmarshal(raw, &v)
	return v.SubagentType, v.Model
}
