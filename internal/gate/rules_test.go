package gate

import (
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

func obs(id, category, signal string) substrate.Observation {
	return substrate.Observation{ID: id, SessionID: "s1", CategoryID: category, SignalType: signal}
}

func obsMode(id, sessionID, category, signal, mode string) substrate.Observation {
	return substrate.Observation{ID: id, SessionID: sessionID, CategoryID: category, SignalType: signal, PermissionMode: mode}
}

func repeat(n int, f func(i int) substrate.Observation) []substrate.Observation {
	out := make([]substrate.Observation, n)
	for i := 0; i < n; i++ {
		out[i] = f(i)
	}
	return out
}

// --- trust_qa ---

func TestEstimateTrustQAInsufficientSignalBelowMinimum(t *testing.T) {
	window := repeat(3, func(i int) substrate.Observation {
		return obs("e", substrate.CategoryTrustQA, substrate.SignalEditRecorded)
	})
	got := estimateTrustQA(window, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v", got.State, StateInsufficientSignal)
	}
}

func TestEstimateTrustQARung1AllUninspected(t *testing.T) {
	var window []substrate.Observation
	for i := 0; i < trustQAMinEvents; i++ {
		window = append(window,
			obs("e", substrate.CategoryTrustQA, substrate.SignalEditRecorded),
			obs("u", substrate.CategoryTrustQA, substrate.SignalEditUninspectedBeforeNext),
		)
	}
	got := estimateTrustQA(window, time.Now())
	if got.State != StateFloor || got.Rung != 1 || got.Confidence != "high" {
		t.Errorf("got %+v, want floor rung 1 high", got)
	}
}

func TestEstimateTrustQARung3ReadOnlyInspection(t *testing.T) {
	var window []substrate.Observation
	for i := 0; i < trustQAMinEvents; i++ {
		window = append(window,
			obs("e", substrate.CategoryTrustQA, substrate.SignalEditRecorded),
			obs("i", substrate.CategoryTrustQA, substrate.SignalInspectionAfterEdit),
		)
	}
	got := estimateTrustQA(window, time.Now())
	if got.State != StateFloor || got.Rung != 3 {
		t.Errorf("got %+v, want floor rung 3", got)
	}
}

func TestEstimateTrustQARung4WithLSP(t *testing.T) {
	var window []substrate.Observation
	for i := 0; i < trustQAMinEvents; i++ {
		window = append(window,
			obs("e", substrate.CategoryTrustQA, substrate.SignalEditRecorded),
			obs("i", substrate.CategoryTrustQA, substrate.SignalInspectionAfterEdit),
		)
	}
	window = append(window, obs("l", substrate.CategoryTrustQA, substrate.SignalLSPDiagnosticsUsed))
	got := estimateTrustQA(window, time.Now())
	if got.State != StateFloor || got.Rung != 4 {
		t.Errorf("got %+v, want floor rung 4", got)
	}
}

func TestEstimateTrustQARung6HighInspectionMultipleVerificationCategories(t *testing.T) {
	var window []substrate.Observation
	for i := 0; i < trustQAMinEvents; i++ {
		window = append(window,
			obs("e", substrate.CategoryTrustQA, substrate.SignalEditRecorded),
			obs("i", substrate.CategoryTrustQA, substrate.SignalInspectionAfterEdit),
		)
	}
	v1 := obs("v1", substrate.CategoryTrustQA, substrate.SignalVerificationCommandRan)
	v1.Detail = map[string]any{"category": "typecheck"}
	v2 := obs("v2", substrate.CategoryTrustQA, substrate.SignalVerificationCommandRan)
	v2.Detail = map[string]any{"category": "test"}
	window = append(window, v1, v2)

	got := estimateTrustQA(window, time.Now())
	if got.State != StateFloor || got.Rung != 6 {
		t.Errorf("got %+v, want floor rung 6", got)
	}
}

func TestEstimateTrustQACandidateSignalsNeverMoveFloor(t *testing.T) {
	var window []substrate.Observation
	for i := 0; i < trustQAMinEvents; i++ {
		window = append(window,
			obs("e", substrate.CategoryTrustQA, substrate.SignalEditRecorded),
			obs("u", substrate.CategoryTrustQA, substrate.SignalEditUninspectedBeforeNext),
		)
	}
	window = append(window,
		obs("sub", substrate.CategoryTrustQA, substrate.SignalSubagentInvoked),
		obs("browser", substrate.CategoryTrustQA, substrate.SignalBrowserOrE2EToolInvoked),
	)
	got := estimateTrustQA(window, time.Now())
	if got.Rung != 1 {
		t.Errorf("candidate signals moved the floor rung: got %+v", got)
	}
	if len(got.CandidateSignals) != 2 {
		t.Fatalf("expected 2 candidate signals, got %+v", got.CandidateSignals)
	}
	byRung := map[int]CandidateSignal{}
	for _, c := range got.CandidateSignals {
		byRung[c.Rung] = c
	}
	if byRung[7].Count != 1 || byRung[8].Count != 1 {
		t.Errorf("candidate signals = %+v, want rung 7 and 8 each count 1", got.CandidateSignals)
	}
}

// --- agent_invocation ---

func TestEstimateAgentInvocationInsufficientSignal(t *testing.T) {
	got := estimateAgentInvocation(nil, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v", got.State, StateInsufficientSignal)
	}
}

func TestEstimateAgentInvocationDefaultMode(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "default"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.State != StateFloor || got.Rung != 1 || got.Confidence != "high" {
		t.Errorf("got %+v, want floor rung 1 high", got)
	}
}

func TestEstimateAgentInvocationDontAskMapsToRung3(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "dontAsk"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.State != StateFloor || got.Rung != 3 || got.Confidence != "medium" {
		t.Errorf("got %+v, want floor rung 3 medium", got)
	}
}

func TestEstimateAgentInvocationAutoModeIsUnmapped(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "auto"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.State != StateUnmapped {
		t.Errorf("State = %v, want %v", got.State, StateUnmapped)
	}
	if len(got.UnmappedValues) != 1 || got.UnmappedValues[0] != "auto" {
		t.Errorf("UnmappedValues = %+v, want [auto]", got.UnmappedValues)
	}
	if got.Rung != 0 {
		t.Errorf("Rung = %d, want 0 (unmapped never claims a rung)", got.Rung)
	}
}

func TestEstimateAgentInvocationBypassPermissionsIsUnmapped(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "bypassPermissions"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.State != StateUnmapped {
		t.Errorf("State = %v, want %v", got.State, StateUnmapped)
	}
}

func TestEstimateAgentInvocationPlanModeIsUnmapped(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "plan"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.State != StateUnmapped {
		t.Errorf("State = %v, want %v", got.State, StateUnmapped)
	}
}

func TestEstimateAgentInvocationAskUserQuestionIsNotedNotSubtracted(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "dontAsk"),
		obsMode("a1", "s1", substrate.CategoryAgentInvocation, substrate.SignalAskUserQuestionInvoked, "dontAsk"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.State != StateFloor || got.Rung != 3 {
		t.Errorf("AskUserQuestion co-occurrence should not block the dontAsk rung-3 read, got %+v", got)
	}
	if got.Notes == "" {
		t.Error("expected a note flagging the AskUserQuestion co-occurrence")
	}
}

func TestEstimateAgentInvocationMajorityModeWins(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "default"),
		obsMode("p2", "s2", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "acceptEdits"),
		obsMode("p3", "s3", substrate.CategoryAgentInvocation, substrate.SignalPermissionModeSnapshot, "acceptEdits"),
	}
	got := estimateAgentInvocation(window, time.Now())
	if got.Rung != 2 {
		t.Errorf("Rung = %d, want 2 (acceptEdits is the majority across sessions)", got.Rung)
	}
}

// --- context_mgmt ---

func TestEstimateContextMgmtInsufficientSignalNoCompaction(t *testing.T) {
	got := estimateContextMgmt(nil, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v", got.State, StateInsufficientSignal)
	}
}

func TestEstimateContextMgmtRung4OnManualCompact(t *testing.T) {
	o := obs("c1", substrate.CategoryContextMgmt, substrate.SignalCompactBoundaryObserved)
	o.Detail = map[string]any{"trigger": "manual"}
	got := estimateContextMgmt([]substrate.Observation{o}, time.Now())
	if got.State != StateFloor || got.Rung != 4 || got.Confidence != "high" {
		t.Errorf("got %+v, want floor rung 4 high", got)
	}
}

func TestEstimateContextMgmtRung1OnRecurringAutoCompact(t *testing.T) {
	var window []substrate.Observation
	for i := 0; i < contextMgmtMinAutoCount; i++ {
		o := obs("c", substrate.CategoryContextMgmt, substrate.SignalCompactBoundaryObserved)
		o.Detail = map[string]any{"trigger": "auto"}
		window = append(window, o)
	}
	got := estimateContextMgmt(window, time.Now())
	if got.State != StateFloor || got.Rung != 1 || got.Confidence != "medium" {
		t.Errorf("got %+v, want floor rung 1 medium", got)
	}
}

func TestEstimateContextMgmtInsufficientSignalSingleAutoCompact(t *testing.T) {
	o := obs("c1", substrate.CategoryContextMgmt, substrate.SignalCompactBoundaryObserved)
	o.Detail = map[string]any{"trigger": "auto"}
	got := estimateContextMgmt([]substrate.Observation{o}, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v (single auto compact is too thin)", got.State, StateInsufficientSignal)
	}
}

// --- prompting_structure ---

func TestEstimatePromptingStructureInsufficientSignal(t *testing.T) {
	window := repeat(3, func(i int) substrate.Observation {
		return obs("t", substrate.CategoryPromptingStructure, substrate.SignalTypedPromptObserved)
	})
	got := estimatePromptingStructure(window, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v", got.State, StateInsufficientSignal)
	}
}

func TestEstimatePromptingStructureRung2PlainPrompts(t *testing.T) {
	window := repeat(promptingMinEvents, func(i int) substrate.Observation {
		return obs("t", substrate.CategoryPromptingStructure, substrate.SignalTypedPromptObserved)
	})
	got := estimatePromptingStructure(window, time.Now())
	if got.State != StateFloor || got.Rung != 2 {
		t.Errorf("got %+v, want floor rung 2", got)
	}
}

func TestEstimatePromptingStructureRung4WithPathOrURL(t *testing.T) {
	window := repeat(promptingMinEvents, func(i int) substrate.Observation {
		return obs("t", substrate.CategoryPromptingStructure, substrate.SignalTypedPromptObserved)
	})
	window = append(window, obs("p", substrate.CategoryPromptingStructure, substrate.SignalPromptContainsPath))
	got := estimatePromptingStructure(window, time.Now())
	if got.State != StateFloor || got.Rung != 4 {
		t.Errorf("got %+v, want floor rung 4", got)
	}
}

func TestEstimatePromptingStructureFloorRung5OnPlanMode(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryPromptingStructure, substrate.SignalPermissionModeSnapshot, "plan"),
	}
	got := estimatePromptingStructure(window, time.Now())
	if got.State != StateFloor || got.Rung != 5 {
		t.Errorf("got %+v, want floor rung 5", got)
	}
	if got.Confidence != "high" {
		t.Errorf("Confidence = %q, want high", got.Confidence)
	}
}

func TestEstimatePromptingStructurePlanModeTakesPriorityOverTypedPrompts(t *testing.T) {
	window := repeat(promptingMinEvents, func(i int) substrate.Observation {
		return obs("t", substrate.CategoryPromptingStructure, substrate.SignalTypedPromptObserved)
	})
	window = append(window, obsMode("p1", "s1", substrate.CategoryPromptingStructure, substrate.SignalPermissionModeSnapshot, "plan"))
	got := estimatePromptingStructure(window, time.Now())
	if got.State != StateFloor || got.Rung != 5 {
		t.Errorf("got %+v, want floor rung 5 (plan-mode evidence outranks typed-prompt evidence)", got)
	}
}

// --- model_routing ---

func TestEstimateModelRoutingInsufficientSignalNoData(t *testing.T) {
	got := estimateModelRouting(nil, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v", got.State, StateInsufficientSignal)
	}
}

func TestEstimateModelRoutingRung1ConstantModel(t *testing.T) {
	window := []substrate.Observation{
		obs("m1", substrate.CategoryModelRouting, substrate.SignalWindowUtilizationSample),
	}
	got := estimateModelRouting(window, time.Now())
	if got.State != StateFloor || got.Rung != 1 {
		t.Errorf("got %+v, want floor rung 1 (default to lower under ambiguity)", got)
	}
}

func TestEstimateModelRoutingBandedOnSwitch(t *testing.T) {
	window := []substrate.Observation{
		obs("sw1", substrate.CategoryModelRouting, substrate.SignalModelSwitchObserved),
	}
	got := estimateModelRouting(window, time.Now())
	if got.State != StateBanded {
		t.Errorf("State = %v, want %v", got.State, StateBanded)
	}
	if got.RungMin != 2 || got.RungMax != 4 {
		t.Errorf("RungMin/RungMax = %d/%d, want 2/4", got.RungMin, got.RungMax)
	}
}

// --- parallelization ---

func TestEstimateParallelizationInsufficientSignalNoData(t *testing.T) {
	got := estimateParallelization(nil, time.Now())
	if got.State != StateInsufficientSignal {
		t.Errorf("State = %v, want %v", got.State, StateInsufficientSignal)
	}
}

func TestEstimateParallelizationBandedOnConcurrency(t *testing.T) {
	window := []substrate.Observation{
		obs("c1", substrate.CategoryParallelization, substrate.SignalConcurrentSessionsObserved),
	}
	got := estimateParallelization(window, time.Now())
	if got.State != StateBanded {
		t.Errorf("State = %v, want %v", got.State, StateBanded)
	}
	if got.RungMin != 5 || got.RungMax != 6 {
		t.Errorf("RungMin/RungMax = %d/%d, want 5/6", got.RungMin, got.RungMax)
	}
	if got.Confidence != "medium" {
		t.Errorf("Confidence = %q, want medium", got.Confidence)
	}
}

func TestEstimateParallelizationFloorRung2OnWorktreeOnly(t *testing.T) {
	window := []substrate.Observation{
		obs("w1", substrate.CategoryParallelization, substrate.SignalGitWorktreeAdded),
	}
	got := estimateParallelization(window, time.Now())
	if got.State != StateFloor || got.Rung != 2 || got.Confidence != "high" {
		t.Errorf("got %+v, want floor rung 2 high", got)
	}
}

func TestEstimateParallelizationFloorRung1WhenNeitherSignal(t *testing.T) {
	window := []substrate.Observation{
		obsMode("p1", "s1", substrate.CategoryParallelization, substrate.SignalPermissionModeSnapshot, "default"),
	}
	got := estimateParallelization(window, time.Now())
	if got.State != StateFloor || got.Rung != 1 || got.Confidence != "medium" {
		t.Errorf("got %+v, want floor rung 1 medium (default to lower under ambiguity)", got)
	}
}

func TestEstimateParallelizationConcurrencyTakesPriorityOverWorktree(t *testing.T) {
	window := []substrate.Observation{
		obs("w1", substrate.CategoryParallelization, substrate.SignalGitWorktreeAdded),
		obs("c1", substrate.CategoryParallelization, substrate.SignalConcurrentSessionsObserved),
	}
	got := estimateParallelization(window, time.Now())
	if got.State != StateBanded {
		t.Errorf("State = %v, want %v (concurrency evidence outranks worktree-only)", got.State, StateBanded)
	}
}
