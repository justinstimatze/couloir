package transcriptscan

import (
	"bufio"
	"encoding/json"
	"io"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// ClassifyResult carries the observations extracted from one scan pass
// plus the checkpoint state to persist for next time.
type ClassifyResult struct {
	Observations []substrate.Observation
	BytesRead    int64
	LastModel    string
}

// Classify reads transcript lines from r — already positioned at the
// prior checkpoint's byte offset, so r only contains new lines — and
// extracts raw facts for context_mgmt, prompting_structure, and
// model_routing. It never errors: an unparseable line, or a recognized
// type with an unexpected shape, is skipped rather than fatal, matching
// the PreToolUse Lens's fail-open discipline. An unrecognized `type`
// (this host runs a heavy hook/plugin stack that injects its own line
// types) is silently ignored, not an error.
func Classify(r io.Reader, prevModel, sessionID string, now time.Time, newID func() string) ClassifyResult {
	result := ClassifyResult{LastModel: prevModel}
	ts := now.UTC().Format(time.RFC3339)

	emit := func(category, signal string, detail map[string]any, modelID string) {
		result.Observations = append(result.Observations, substrate.Observation{
			SchemaVersion: substrate.SchemaVersion,
			ID:            newID(),
			ObservedAt:    ts,
			SessionID:     sessionID,
			Lens:          substrate.LensTranscriptScan,
			CategoryID:    category,
			SignalType:    signal,
			Detail:        detail,
			ModelID:       modelID,
		})
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // transcript lines can carry long tool output
	for sc.Scan() {
		line := sc.Bytes()
		result.BytesRead += int64(len(line)) + 1 // +1 for the newline bufio.Scanner strips

		var env transcriptLine
		if json.Unmarshal(line, &env) != nil {
			continue
		}

		switch env.Type {
		case "assistant":
			classifyAssistantLine(env, &result, emit)
		case "system":
			classifySystemLine(env, emit)
		case "user":
			classifyUserLine(env, emit)
		}
	}

	return result
}

func classifyAssistantLine(env transcriptLine, result *ClassifyResult, emit func(category, signal string, detail map[string]any, modelID string)) {
	var am assistantMessage
	if json.Unmarshal(env.Message, &am) != nil || am.Model == "" {
		return
	}
	if result.LastModel != "" && am.Model != result.LastModel {
		emit(substrate.CategoryModelRouting, substrate.SignalModelSwitchObserved, map[string]any{
			"previous_model": result.LastModel,
			"new_model":      am.Model,
		}, am.Model)
	}
	result.LastModel = am.Model

	total := am.Usage.InputTokens + am.Usage.CacheCreationInputTokens + am.Usage.CacheReadInputTokens + am.Usage.OutputTokens
	if total > 0 {
		limit := contextLimitFor(am.Model)
		emit(substrate.CategoryContextMgmt, substrate.SignalWindowUtilizationSample, map[string]any{
			"total_tokens":    total,
			"context_limit":   limit,
			"utilization_pct": float64(total) / float64(limit) * 100,
		}, am.Model)
	}
}

func classifySystemLine(env transcriptLine, emit func(category, signal string, detail map[string]any, modelID string)) {
	if env.Subtype != "compact_boundary" || env.CompactMetadata == nil {
		return
	}
	emit(substrate.CategoryContextMgmt, substrate.SignalCompactBoundaryObserved, map[string]any{
		"trigger": env.CompactMetadata.Trigger,
	}, "")
}

func classifyUserLine(env transcriptLine, emit func(category, signal string, detail map[string]any, modelID string)) {
	if env.PromptSource != "typed" {
		return
	}
	var um userMessage
	if json.Unmarshal(env.Message, &um) != nil {
		return
	}
	var text string
	if json.Unmarshal(um.Content, &text) != nil || text == "" {
		return // array-shaped content (tool_result etc.) is not a typed prompt
	}

	emit(substrate.CategoryPromptingStructure, substrate.SignalTypedPromptObserved, nil, "")
	if containsAbsolutePath(text) {
		emit(substrate.CategoryPromptingStructure, substrate.SignalPromptContainsPath, nil, "")
	}
	if containsURL(text) {
		emit(substrate.CategoryPromptingStructure, substrate.SignalPromptContainsURL, nil, "")
	}
}
