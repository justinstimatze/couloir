package gate

import (
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// Ripeness constants. The unit differs by category shape — session-count
// for signals that are near-constant within a session (permission mode,
// model choice), rolling observation-count for signals that are dense
// per-tool-call (edits, prompts). Defaults, not measured — they need
// tuning against real observations.jsonl data over time, not precision
// engineering up front.
const (
	trustQAWindowCount         = 30
	trustQAMinEvents           = 10
	promptingWindowCount       = 30
	promptingMinEvents         = 10
	agentInvocationSessions    = 3
	modelRoutingSessions       = 3
	contextMgmtSessions        = 5
	contextMgmtMinAutoCount    = 2
	parallelizationSessions    = 5
	promptingStructureSessions = 3
)

func estimateTrustQA(obs []substrate.Observation, now time.Time) RungEstimate {
	cat := filterCategory(obs, substrate.CategoryTrustQA)
	window := lastNByCount(cat, trustQAWindowCount)
	edits := filterSignal(window, substrate.SignalEditRecorded)
	est := RungEstimate{CategoryID: substrate.CategoryTrustQA, AsOf: now}
	est.CandidateSignals = trustQACandidateSignals(window)

	if len(edits) < trustQAMinEvents {
		est.State = StateInsufficientSignal
		est.Notes = "fewer than the minimum edit_recorded events observed in window"
		return est
	}

	uninspected := filterSignal(window, substrate.SignalEditUninspectedBeforeNext)
	inspected := filterSignal(window, substrate.SignalInspectionAfterEdit)

	if len(uninspected) > 0 && len(inspected) == 0 {
		est.State = StateFloor
		est.Rung = 1
		est.Confidence = "high"
		est.ObservationIDs = ids(uninspected)
		return est
	}

	if len(inspected) == 0 {
		est.State = StateInsufficientSignal
		est.Notes = "edits observed but none yet resolved by an inspection or an abandonment"
		return est
	}

	lsp := filterSignal(window, substrate.SignalLSPDiagnosticsUsed)
	verification := filterSignal(window, substrate.SignalVerificationCommandRan)
	typecheck := verificationByCategory(verification, "typecheck")
	distinctVerifCategories := distinctVerificationCategories(verification)

	hasRung4Evidence := len(lsp) > 0 || len(typecheck) > 0
	inspectionRate := float64(len(inspected)) / float64(len(inspected)+len(uninspected))

	switch {
	case hasRung4Evidence && distinctVerifCategories >= 2 && inspectionRate >= 0.8:
		est.State = StateFloor
		est.Rung = 6
		est.Confidence = "medium"
		est.ObservationIDs = append(ids(inspected), ids(verification)...)
	case hasRung4Evidence:
		est.State = StateFloor
		est.Rung = 4
		est.Confidence = "high"
		est.ObservationIDs = append(ids(lsp), ids(typecheck)...)
	default:
		est.State = StateFloor
		est.Rung = 3
		est.Confidence = "medium"
		est.ObservationIDs = ids(inspected)
	}
	return est
}

func trustQACandidateSignals(window []substrate.Observation) []CandidateSignal {
	var out []CandidateSignal
	if sub := filterSignal(window, substrate.SignalSubagentInvoked); len(sub) > 0 {
		out = append(out, CandidateSignal{
			Rung: 7, Count: len(sub), ObservationIDs: ids(sub),
			Caveat: "an Agent tool call happened; no purpose classification available, could be anything",
		})
	}
	if browser := filterSignal(window, substrate.SignalBrowserOrE2EToolInvoked); len(browser) > 0 {
		out = append(out, CandidateSignal{
			Rung: 8, Count: len(browser), ObservationIDs: ids(browser),
			Caveat: "a browser/e2e tool call happened; no purpose classification available, could be anything",
		})
	}
	return out
}

func verificationByCategory(verification []substrate.Observation, category string) []substrate.Observation {
	var out []substrate.Observation
	for _, o := range verification {
		if c, _ := o.Detail["category"].(string); c == category {
			out = append(out, o)
		}
	}
	return out
}

func distinctVerificationCategories(verification []substrate.Observation) int {
	seen := map[string]bool{}
	for _, o := range verification {
		if c, _ := o.Detail["category"].(string); c != "" {
			seen[c] = true
		}
	}
	return len(seen)
}

func estimateAgentInvocation(obs []substrate.Observation, now time.Time) RungEstimate {
	cat := filterCategory(obs, substrate.CategoryAgentInvocation)
	window := lastNSessions(cat, agentInvocationSessions)
	snapshots := filterSignal(window, substrate.SignalPermissionModeSnapshot)

	est := RungEstimate{CategoryID: substrate.CategoryAgentInvocation, AsOf: now}
	if askCount := filterSignal(window, substrate.SignalAskUserQuestionInvoked); len(askCount) > 0 {
		est.Notes = "AskUserQuestion observed during this window — counter-evidence for rung 3's suppressed-questions definition, not supporting evidence; does not change the rung reported here"
	}

	if len(snapshots) == 0 {
		est.State = StateInsufficientSignal
		return est
	}

	counts := map[string]int{}
	for _, o := range snapshots {
		counts[o.PermissionMode]++
	}
	mode, modeCount := "", 0
	for m, c := range counts {
		if c > modeCount {
			mode, modeCount = m, c
		}
	}

	switch mode {
	case "default":
		est.State, est.Rung, est.Confidence = StateFloor, 1, "high"
	case "acceptEdits":
		est.State, est.Rung, est.Confidence = StateFloor, 2, "high"
	case "dontAsk":
		est.State, est.Rung, est.Confidence = StateFloor, 3, "medium"
	default:
		est.State = StateUnmapped
		est.UnmappedValues = []string{mode}
	}
	est.ObservationIDs = ids(observationsWithMode(snapshots, mode))
	return est
}

func observationsWithMode(obs []substrate.Observation, mode string) []substrate.Observation {
	var out []substrate.Observation
	for _, o := range obs {
		if o.PermissionMode == mode {
			out = append(out, o)
		}
	}
	return out
}

func estimateContextMgmt(obs []substrate.Observation, now time.Time) RungEstimate {
	cat := filterCategory(obs, substrate.CategoryContextMgmt)
	window := lastNSessions(cat, contextMgmtSessions)
	compacts := filterSignal(window, substrate.SignalCompactBoundaryObserved)

	est := RungEstimate{CategoryID: substrate.CategoryContextMgmt, AsOf: now}
	if len(compacts) == 0 {
		est.State = StateInsufficientSignal
		return est
	}

	var manual, auto []substrate.Observation
	for _, o := range compacts {
		if t, _ := o.Detail["trigger"].(string); t == "manual" {
			manual = append(manual, o)
		} else {
			auto = append(auto, o)
		}
	}

	switch {
	case len(manual) > 0:
		est.State, est.Rung, est.Confidence = StateFloor, 4, "high"
		est.ObservationIDs = ids(manual)
	case len(auto) >= contextMgmtMinAutoCount:
		est.State, est.Rung, est.Confidence = StateFloor, 1, "medium"
		est.ObservationIDs = ids(auto)
	default:
		est.State = StateInsufficientSignal
		est.Notes = "only one automatic compaction observed — too thin to call a recurring pattern"
	}
	return est
}

func estimatePromptingStructure(obs []substrate.Observation, now time.Time) RungEstimate {
	cat := filterCategory(obs, substrate.CategoryPromptingStructure)

	est := RungEstimate{CategoryID: substrate.CategoryPromptingStructure, AsOf: now}

	// Plan-mode is near-constant-per-session, like agent_invocation's
	// permission-mode signal, so it's windowed and checked the same way
	// -- session count, majority mode -- and takes priority over the
	// typed-prompt evidence below: it's the strongest floor this
	// category has, the same "check strongest evidence first" order
	// estimateParallelization already uses.
	planWindow := lastNSessions(cat, promptingStructureSessions)
	if snapshots := filterSignal(planWindow, substrate.SignalPermissionModeSnapshot); len(snapshots) > 0 {
		counts := map[string]int{}
		for _, o := range snapshots {
			counts[o.PermissionMode]++
		}
		mode, modeCount := "", 0
		for m, c := range counts {
			if c > modeCount {
				mode, modeCount = m, c
			}
		}
		if mode == "plan" {
			est.State, est.Rung, est.Confidence = StateFloor, 5, "high"
			est.ObservationIDs = ids(observationsWithMode(snapshots, mode))
			return est
		}
	}

	window := lastNByCount(cat, promptingWindowCount)
	typed := filterSignal(window, substrate.SignalTypedPromptObserved)
	if len(typed) < promptingMinEvents {
		est.State = StateInsufficientSignal
		est.Notes = "fewer than the minimum typed_prompt_observed events observed in window"
		return est
	}

	paths := filterSignal(window, substrate.SignalPromptContainsPath)
	urls := filterSignal(window, substrate.SignalPromptContainsURL)

	if len(paths) > 0 || len(urls) > 0 {
		est.State, est.Rung, est.Confidence = StateFloor, 4, "high"
		est.ObservationIDs = append(ids(paths), ids(urls)...)
		return est
	}
	est.State, est.Rung, est.Confidence = StateFloor, 2, "medium"
	est.ObservationIDs = ids(typed)
	return est
}

func estimateModelRouting(obs []substrate.Observation, now time.Time) RungEstimate {
	cat := filterCategory(obs, substrate.CategoryModelRouting)
	window := lastNSessions(cat, modelRoutingSessions)

	est := RungEstimate{CategoryID: substrate.CategoryModelRouting, AsOf: now}
	if len(window) == 0 {
		est.State = StateInsufficientSignal
		return est
	}

	switches := filterSignal(window, substrate.SignalModelSwitchObserved)
	if len(switches) > 0 {
		est.State = StateBanded
		est.RungMin, est.RungMax = 2, 4
		est.Confidence = "low"
		est.ObservationIDs = ids(switches)
		est.Notes = "a raw model-id change can't distinguish ad hoc switching, task-type matching, or router delegation"
		return est
	}

	// Constant model observed, no switch: default to the lower rung
	// under ambiguity (rung 5's deliberate-largest-model-policy isn't
	// distinguishable from rung 1's never-decided from silence alone).
	est.State, est.Rung, est.Confidence = StateFloor, 1, "medium"
	est.ObservationIDs = ids(window)
	return est
}

func estimateParallelization(obs []substrate.Observation, now time.Time) RungEstimate {
	cat := filterCategory(obs, substrate.CategoryParallelization)
	window := lastNSessions(cat, parallelizationSessions)

	est := RungEstimate{CategoryID: substrate.CategoryParallelization, AsOf: now}
	if len(window) == 0 {
		est.State = StateInsufficientSignal
		return est
	}

	if concurrent := filterSignal(window, substrate.SignalConcurrentSessionsObserved); len(concurrent) > 0 {
		est.State = StateBanded
		est.RungMin, est.RungMax = 5, 6
		est.Confidence = "medium"
		est.ObservationIDs = ids(concurrent)
		est.Notes = "concurrent session activity observed -- can't distinguish an ad hoc burst from a deliberate numeric-scaling discipline from file activity alone"
		return est
	}

	if worktree := filterSignal(window, substrate.SignalGitWorktreeAdded); len(worktree) > 0 {
		est.State, est.Rung, est.Confidence = StateFloor, 2, "high"
		est.ObservationIDs = ids(worktree)
		return est
	}

	// Neither signal: default to the lower rung under ambiguity, same as
	// model_routing's constant-model floor -- a session run alone
	// doesn't prove parallel execution was never considered, only that
	// it wasn't observed.
	est.State, est.Rung, est.Confidence = StateFloor, 1, "medium"
	est.ObservationIDs = ids(window)
	return est
}
