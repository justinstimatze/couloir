package reasoner

import (
	"github.com/justinstimatze/couloir/internal/gate"
	"github.com/justinstimatze/couloir/internal/substrate"
)

// RungCitation is a rung's definition plus the provenance the corpus
// already carries for it — never a fabricated summary. Matches the
// corpus's own IP-handling convention: short attributed excerpt, full
// source shown alongside it.
type RungCitation struct {
	Rung       int    `json:"rung"`
	Definition string `json:"definition"`
	SourceName string `json:"source_name"`
	SourceURL  string `json:"source_url"`
	Excerpt    string `json:"excerpt"`
}

// Suggestion is one category's optional next-step, framed exactly per
// HANDOFF.md's "What this is" section: the next rung up, and separately
// a frontier example, both explicitly optional/fyi, never a ceiling
// comparison.
type Suggestion struct {
	CategoryID  string        `json:"category_id"`
	CurrentRung int           `json:"current_rung,omitempty"`
	Confidence  string        `json:"confidence"`
	NextRung    *RungCitation `json:"next_rung,omitempty"`
	Frontier    *RungCitation `json:"frontier,omitempty"`
	Caution     *Caution      `json:"caution,omitempty"`
}

// Caution is a cross-cutting note tied to a category's observed state
// rather than to a specific rung — matches the corpus's own
// rung_number: null rows (meta findings that don't fit a single rung).
// No Rung field, deliberately: a caution never claims a place on the
// ladder.
type Caution struct {
	Definition string `json:"definition"`
	SourceName string `json:"source_name"`
	SourceURL  string `json:"source_url"`
	Excerpt    string `json:"excerpt"`
}

// parallelizationCautionRowID is the corpus's own cost/spend caution for
// concurrent multi-agent sessions (rung_number: null in data/corpus.jsonl
// — a cross-cutting note, not evidence for a rung). Hardcoded rather than
// generalized into a per-category caution mechanism: this is the one
// case that exists right now.
const parallelizationCautionRowID = "d026c6e0-e175-4c72-8229-1ff5c373fab5"

// Suggest produces a deterministic, cited suggestion per category with
// a confident single-rung floor. Categories in any other state
// (insufficient_signal, banded, unmapped) get no suggestion — a rung
// range or an unmapped value has no single "current rung" to add one
// to, and forcing one would be exactly the fabrication this project
// rejects everywhere else.
func Suggest(estimates []gate.RungEstimate, ladder *Ladder, corpus map[string]CorpusRow) []Suggestion {
	var out []Suggestion
	for _, est := range estimates {
		if est.CategoryID == substrate.CategoryParallelization && est.State == gate.StateBanded {
			row, ok := corpus[parallelizationCautionRowID]
			if !ok {
				continue
			}
			s := Suggestion{
				CategoryID: est.CategoryID,
				Confidence: "medium",
				Caution: &Caution{
					Definition: row.Practice,
					SourceName: row.SourceName,
					SourceURL:  row.SourceURL,
					Excerpt:    row.Excerpt,
				},
			}
			// The caution names a real cost trade-off, but a banded
			// estimate (concurrent activity, ambiguous rung 5-6) sits
			// right below where this ladder actually aspires to --
			// deliberate scaling, cloud-hosted concurrency, dozens of
			// agents via role-splitting. Pairing the caution with that
			// frontier, same as every StateFloor suggestion gets, keeps
			// it a trade-off to weigh rather than pure friction with no
			// destination shown.
			if cat := ladder.CategoryByID(est.CategoryID); cat != nil {
				if frontier := cat.HighestPopulatedRung(); frontier != nil && frontier.Rung > est.RungMax {
					s.Frontier = citeRung(frontier, corpus)
				}
			}
			out = append(out, s)
			continue
		}
		if est.State != gate.StateFloor {
			continue
		}
		cat := ladder.CategoryByID(est.CategoryID)
		if cat == nil {
			continue
		}

		s := Suggestion{CategoryID: est.CategoryID, CurrentRung: est.Rung, Confidence: est.Confidence}
		if next := cat.RungByNumber(est.Rung + 1); next != nil {
			s.NextRung = citeRung(next, corpus)
		}
		if frontier := cat.HighestPopulatedRung(); frontier != nil && frontier.Rung > est.Rung+1 {
			s.Frontier = citeRung(frontier, corpus)
		}
		if s.NextRung == nil && s.Frontier == nil {
			continue // rung 8 floor, or the next cell has no ladder definition -- nothing to suggest
		}
		out = append(out, s)
	}
	return out
}

// citeRung builds a citation from a rung's first evidence row. A rung
// with no definition, or no resolvable evidence row, yields nil rather
// than a citation with missing provenance.
func citeRung(r *Rung, corpus map[string]CorpusRow) *RungCitation {
	if r.Definition == nil || len(r.EvidenceRowIDs) == 0 {
		return nil
	}
	row, ok := corpus[r.EvidenceRowIDs[0]]
	if !ok {
		return nil
	}
	return &RungCitation{
		Rung:       r.Rung,
		Definition: *r.Definition,
		SourceName: row.SourceName,
		SourceURL:  row.SourceURL,
		Excerpt:    row.Excerpt,
	}
}
