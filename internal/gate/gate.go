// Package gate is the one place in this system allowed to turn raw
// substrate facts into a rung number — deterministic aggregation, never
// an LLM call, never a fabricated guess. Every field is documented
// against data/ladder.json's and data/corpus.jsonl's actual text (see
// rules.go), not a vibe read, and every rung it does claim carries the
// substrate observation ids that justified it.
package gate

import (
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// RungState is what kind of answer Gate is giving for a category — a
// confident rung is only one of several honest outcomes.
type RungState string

const (
	// StateFloor: a specific rung, backed by ObservationIDs.
	StateFloor RungState = "floor"
	// StateBanded: the raw signal can't distinguish between adjacent
	// rungs (e.g. a model switch alone can't tell ad hoc from
	// router-delegated) — RungMin/RungMax carry the range.
	StateBanded RungState = "banded"
	// StateInsufficientSignal: the ladder has a definition here, but
	// this window hasn't accumulated enough observations to say
	// anything yet.
	StateInsufficientSignal RungState = "insufficient_signal"
	// StateUnmapped: a real, observed value that doesn't sit cleanly
	// on this category's 1-8 line at all (e.g. permission_mode
	// "auto"/"bypassPermissions"/"plan") — reported, not forced onto a
	// rung it doesn't support.
	StateUnmapped RungState = "unmapped"
)

// CandidateSignal is a mechanical fact that COULD be weak evidence for a
// higher rung but carries no purpose classification (an Agent tool call
// could be for anything). Never moves the floor rung, never counts
// toward ripeness — kept visible and cited, not silently discarded.
type CandidateSignal struct {
	Rung           int      `json:"rung"`
	Count          int      `json:"count"`
	ObservationIDs []string `json:"observation_ids"`
	Caveat         string   `json:"caveat"`
}

// RungEstimate is Gate's answer for one category.
type RungEstimate struct {
	CategoryID       string            `json:"category_id"`
	State            RungState         `json:"state"`
	Rung             int               `json:"rung,omitempty"`     // StateFloor only
	RungMin          int               `json:"rung_min,omitempty"` // StateBanded only
	RungMax          int               `json:"rung_max,omitempty"` // StateBanded only
	Confidence       string            `json:"confidence,omitempty"`
	ObservationIDs   []string          `json:"observation_ids,omitempty"`
	CandidateSignals []CandidateSignal `json:"candidate_signals,omitempty"`
	UnmappedValues   []string          `json:"unmapped_values,omitempty"` // StateUnmapped only
	Notes            string            `json:"notes,omitempty"`
	AsOf             time.Time         `json:"as_of"`
}

// Estimate computes one RungEstimate per category Gate covers, from the
// full set of substrate observations available. Callers should already
// have limited obs to a reasonable tail (Gate itself windows further
// per-category, but reading the whole file on every UserPromptSubmit
// call is a needless cost this function doesn't try to hide).
func Estimate(obs []substrate.Observation, now time.Time) []RungEstimate {
	return []RungEstimate{
		estimateTrustQA(obs, now),
		estimateAgentInvocation(obs, now),
		estimateContextMgmt(obs, now),
		estimatePromptingStructure(obs, now),
		estimateModelRouting(obs, now),
		estimateParallelization(obs, now),
	}
}
