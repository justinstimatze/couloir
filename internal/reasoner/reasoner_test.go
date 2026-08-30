package reasoner

import (
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/gate"
)

func strp(s string) *string { return &s }

func testLadder() *Ladder {
	return &Ladder{
		Categories: []Category{
			{
				CategoryID: "trust_qa",
				Rungs: []Rung{
					{Rung: 1, Definition: strp("rung 1 def"), EvidenceRowIDs: []string{"r1"}},
					{Rung: 2, Definition: nil},
					{Rung: 3, Definition: strp("rung 3 def"), EvidenceRowIDs: []string{"r3"}},
					{Rung: 4, Definition: strp("rung 4 def"), EvidenceRowIDs: []string{"r4"}},
					{Rung: 5, Definition: nil},
					{Rung: 6, Definition: nil},
					{Rung: 7, Definition: nil},
					{Rung: 8, Definition: strp("rung 8 def"), EvidenceRowIDs: []string{"r8"}},
				},
			},
			{
				CategoryID: "parallelization",
				Rungs: []Rung{
					{Rung: 1, Definition: nil},
					{Rung: 2, Definition: nil},
					{Rung: 3, Definition: nil},
					{Rung: 4, Definition: nil},
					{Rung: 5, Definition: nil},
					{Rung: 6, Definition: nil},
					{Rung: 7, Definition: nil},
					{Rung: 8, Definition: strp("parallelization rung 8 def"), EvidenceRowIDs: []string{"p8"}},
				},
			},
		},
	}
}

func testCorpus() map[string]CorpusRow {
	return map[string]CorpusRow{
		"r1": {ID: "r1", SourceName: "Source One", SourceURL: "https://example.com/1", Excerpt: "excerpt one"},
		"r3": {ID: "r3", SourceName: "Source Three", SourceURL: "https://example.com/3", Excerpt: "excerpt three"},
		"r4": {ID: "r4", SourceName: "Source Four", SourceURL: "https://example.com/4", Excerpt: "excerpt four"},
		"r8": {ID: "r8", SourceName: "Source Eight", SourceURL: "https://example.com/8", Excerpt: "excerpt eight"},
		"p8": {ID: "p8", SourceName: "Source P8", SourceURL: "https://example.com/p8", Excerpt: "excerpt p8"},
	}
}

func TestSuggestFloorRung3GetsNextAndFrontier(t *testing.T) {
	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 3, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus())
	if len(got) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(got))
	}
	s := got[0]
	if s.NextRung == nil || s.NextRung.Rung != 4 {
		t.Errorf("NextRung = %+v, want rung 4", s.NextRung)
	}
	if s.Frontier == nil || s.Frontier.Rung != 8 {
		t.Errorf("Frontier = %+v, want rung 8", s.Frontier)
	}
}

func TestSuggestSkipsNextRungWithNoDefinition(t *testing.T) {
	// rung 1's floor puts next-rung at rung 2, which has definition:nil
	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 1, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus())
	if len(got) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(got))
	}
	if got[0].NextRung != nil {
		t.Errorf("NextRung = %+v, want nil (rung 2 has no definition)", got[0].NextRung)
	}
	if got[0].Frontier == nil || got[0].Frontier.Rung != 8 {
		t.Errorf("Frontier = %+v, want rung 8", got[0].Frontier)
	}
}

func TestSuggestFrontierOmittedWhenSameAsNextRung(t *testing.T) {
	// floor at rung 7: next rung is 8, and frontier is also 8 -- must
	// not duplicate the same cell as both NextRung and Frontier.
	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 7, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus())
	if len(got) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(got))
	}
	if got[0].NextRung == nil || got[0].NextRung.Rung != 8 {
		t.Errorf("NextRung = %+v, want rung 8", got[0].NextRung)
	}
	if got[0].Frontier != nil {
		t.Errorf("Frontier = %+v, want nil (same cell as NextRung)", got[0].Frontier)
	}
}

func TestSuggestNoneAtRung8Floor(t *testing.T) {
	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateFloor, Rung: 8, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus())
	if len(got) != 0 {
		t.Errorf("got %+v, want no suggestions at rung 8 (nothing higher to suggest)", got)
	}
}

func TestSuggestSkipsNonFloorStates(t *testing.T) {
	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateInsufficientSignal, AsOf: time.Now()},
		{CategoryID: "trust_qa", State: gate.StateBanded, RungMin: 2, RungMax: 4, AsOf: time.Now()},
		{CategoryID: "trust_qa", State: gate.StateUnmapped, UnmappedValues: []string{"auto"}, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus())
	if len(got) != 0 {
		t.Errorf("got %+v, want no suggestions for non-floor states", got)
	}
}

func TestSuggestSkipsUnknownCategory(t *testing.T) {
	estimates := []gate.RungEstimate{
		{CategoryID: "not_a_real_category", State: gate.StateFloor, Rung: 1, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus())
	if len(got) != 0 {
		t.Errorf("got %+v, want no suggestions for a category absent from the ladder", got)
	}
}

// --- parallelization caution tie-in ---

func testCorpusWithCaution() map[string]CorpusRow {
	c := testCorpus()
	c[parallelizationCautionRowID] = CorpusRow{
		ID:         parallelizationCautionRowID,
		CategoryID: "parallelization",
		Practice:   "COST CAUTION: parallel multi-agent sessions can burn through a plan's usage limits fast.",
		SourceName: "ecliptik (HN handle)",
		SourceURL:  "https://news.ycombinator.com/item?id=47221592",
	}
	return c
}

func TestSuggestBandedParallelizationYieldsCautionAndFrontier(t *testing.T) {
	estimates := []gate.RungEstimate{
		{CategoryID: "parallelization", State: gate.StateBanded, RungMin: 5, RungMax: 6, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpusWithCaution())
	if len(got) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(got))
	}
	s := got[0]
	if s.Caution == nil {
		t.Fatal("expected a Caution")
	}
	if s.Caution.SourceName != "ecliptik (HN handle)" {
		t.Errorf("Caution.SourceName = %q, want ecliptik (HN handle)", s.Caution.SourceName)
	}
	if s.NextRung != nil {
		t.Errorf("a banded estimate must never claim NextRung (no single current rung to add one to), got %+v", s.NextRung)
	}
	if s.Frontier == nil || s.Frontier.Rung != 8 {
		t.Errorf("Frontier = %+v, want rung 8 -- the caution should be paired with where this ladder actually leads, not left as pure friction", s.Frontier)
	}
	if s.CurrentRung != 0 {
		t.Errorf("CurrentRung = %d, want 0 (a banded estimate claims no rung)", s.CurrentRung)
	}
}

func TestSuggestBandedParallelizationOmitsFrontierWhenNothingHigher(t *testing.T) {
	// A ladder whose highest populated parallelization rung sits inside
	// (or below) the banded range itself must not cite it as a
	// "frontier" -- same non-duplication rule the floor branch applies.
	ladder := &Ladder{Categories: []Category{
		{
			CategoryID: "parallelization",
			Rungs: []Rung{
				{Rung: 5, Definition: strp("rung 5 def"), EvidenceRowIDs: []string{"p5"}},
				{Rung: 6, Definition: nil},
			},
		},
	}}
	corpus := testCorpusWithCaution()
	corpus["p5"] = CorpusRow{ID: "p5", SourceName: "Source P5", SourceURL: "https://example.com/p5"}

	estimates := []gate.RungEstimate{
		{CategoryID: "parallelization", State: gate.StateBanded, RungMin: 5, RungMax: 6, AsOf: time.Now()},
	}
	got := Suggest(estimates, ladder, corpus)
	if len(got) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(got))
	}
	if got[0].Frontier != nil {
		t.Errorf("Frontier = %+v, want nil (highest populated rung 5 sits inside the banded range)", got[0].Frontier)
	}
}

func TestSuggestBandedOtherCategoryStillYieldsNoSuggestion(t *testing.T) {
	// Banded on any category other than parallelization must behave
	// exactly as before this change: no suggestion.
	estimates := []gate.RungEstimate{
		{CategoryID: "trust_qa", State: gate.StateBanded, RungMin: 2, RungMax: 4, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpusWithCaution())
	if len(got) != 0 {
		t.Errorf("got %+v, want no suggestions for a banded non-parallelization category", got)
	}
}

func TestSuggestBandedParallelizationMissingCautionRowYieldsNothing(t *testing.T) {
	estimates := []gate.RungEstimate{
		{CategoryID: "parallelization", State: gate.StateBanded, RungMin: 5, RungMax: 6, AsOf: time.Now()},
	}
	got := Suggest(estimates, testLadder(), testCorpus()) // no caution row in this corpus
	if len(got) != 0 {
		t.Errorf("got %+v, want no suggestion when the caution row can't be resolved", got)
	}
}
