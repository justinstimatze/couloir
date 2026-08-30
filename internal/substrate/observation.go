// Package substrate holds the append-only observation log Lens writes to
// and Gate will eventually read from. Every field is a literal fact about
// one tool call — never a rung guess. Rung inference is Gate's job, not
// recorded here.
package substrate

// SchemaVersion tags every row this package writes.
const SchemaVersion = "v0"

// Category values, matching data/ladder.json's category_id vocabulary.
const (
	CategoryTrustQA            = "trust_qa"
	CategoryAgentInvocation    = "agent_invocation"
	CategoryContextMgmt        = "context_mgmt"
	CategoryPromptingStructure = "prompting_structure"
	CategoryModelRouting       = "model_routing"
	CategoryParallelization    = "parallelization"
)

// Lens values, naming which observing mechanism produced a row.
const (
	LensPreToolUse     = "pretooluse"
	LensTranscriptScan = "transcript_scan"
)

// SignalType values this build's PreToolUse Lens emits. See HANDOFF.md's
// Architecture section for which rungs each is candidate evidence for —
// that mapping lives in commentary, never in a stored field.
const (
	SignalPermissionModeSnapshot     = "permission_mode_snapshot"
	SignalAskUserQuestionInvoked     = "ask_user_question_invoked"
	SignalEditRecorded               = "edit_recorded"
	SignalEditUninspectedBeforeNext  = "edit_uninspected_before_next_edit"
	SignalInspectionAfterEdit        = "inspection_after_edit"
	SignalVerificationCommandRan     = "verification_command_ran"
	SignalLSPDiagnosticsUsed         = "lsp_diagnostics_used"
	SignalSubagentInvoked            = "subagent_invoked"
	SignalBrowserOrE2EToolInvoked    = "browser_or_e2e_tool_invoked"
	SignalGitWorktreeAdded           = "git_worktree_added"
	SignalConcurrentSessionsObserved = "concurrent_sessions_observed"
)

// SignalType values the transcript-scan Lens emits. Same discipline as
// above: a raw fact about what the transcript literally contains, never
// a rung guess.
const (
	SignalCompactBoundaryObserved = "compact_boundary_observed"
	SignalWindowUtilizationSample = "window_utilization_sample"
	SignalTypedPromptObserved     = "typed_prompt_observed"
	SignalPromptContainsPath      = "prompt_contains_absolute_path"
	SignalPromptContainsURL       = "prompt_contains_url"
	SignalModelSwitchObserved     = "model_switch_observed"
)

// Observation is one row of the substrate: a typed, literally-true fact
// about a single tool call or transcript line. model_id is populated by
// the transcript-scan Lens (message.model on the relevant transcript
// line) and always empty for the PreToolUse Lens, which has no model
// field on its own hook payload.
type Observation struct {
	SchemaVersion  string         `json:"schema_version"`
	ID             string         `json:"id"`
	ObservedAt     string         `json:"observed_at"`
	SessionID      string         `json:"session_id"`
	Lens           string         `json:"lens"`
	CategoryID     string         `json:"category_id"`
	SignalType     string         `json:"signal_type"`
	ToolName       string         `json:"tool_name"`
	ToolUseID      string         `json:"tool_use_id"`
	PermissionMode string         `json:"permission_mode"`
	SubjectPath    string         `json:"subject_path"`
	Detail         map[string]any `json:"detail"`
	ModelID        string         `json:"model_id,omitempty"`
	Notes          string         `json:"notes,omitempty"`
}
