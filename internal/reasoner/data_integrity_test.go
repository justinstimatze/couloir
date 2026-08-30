package reasoner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/justinstimatze/couloir/data"
)

// fullCorpusRow parses every field this file's checks need -- richer
// than the runtime CorpusRow type, which only carries what Reasoner
// actually cites at runtime. Kept test-local rather than growing the
// production struct for a check nothing at runtime needs.
type fullCorpusRow struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	RungNumber *int   `json:"rung_number"`
	SourceName string `json:"source_name"`
	SourceURL  string `json:"source_url"`
}

func loadFullCorpus(t *testing.T) (rows map[string]fullCorpusRow, lineCount int) {
	t.Helper()
	f, err := data.FS.Open("corpus.jsonl")
	if err != nil {
		t.Fatalf("open corpus.jsonl: %v", err)
	}
	defer f.Close()

	rows = map[string]fullCorpusRow{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lineCount++
		var r fullCorpusRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("corpus.jsonl line %d: malformed JSON: %v", lineCount, err)
		}
		if r.ID == "" {
			t.Fatalf("corpus.jsonl line %d: empty id", lineCount)
		}
		rows[r.ID] = r
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan corpus.jsonl: %v", err)
	}
	return rows, lineCount
}

// TestCorpusHasNoDuplicateIDs catches a bug LoadCorpus's own
// map-keyed-by-id loader would otherwise hide silently: a duplicate id
// overwrites the earlier row in the map with no error, so a stray
// duplicate just quietly drops a real citation.
func TestCorpusHasNoDuplicateIDs(t *testing.T) {
	rows, lineCount := loadFullCorpus(t)
	if len(rows) != lineCount {
		t.Errorf("corpus.jsonl has %d lines but only %d distinct ids -- a duplicate id is silently overwriting an earlier row", lineCount, len(rows))
	}
}

// TestCorpusRowsHaveRealProvenance guards the project's own core claim
// (README: "every claim traces back to a real, attributed source") at
// the data level, not just the code level -- a row missing either
// field would produce a citation with no real attribution.
func TestCorpusRowsHaveRealProvenance(t *testing.T) {
	rows, _ := loadFullCorpus(t)
	for id, r := range rows {
		if r.SourceName == "" {
			t.Errorf("corpus row %s has no source_name", id)
		}
		if r.SourceURL == "" {
			t.Errorf("corpus row %s has no source_url", id)
		}
	}
}

// TestLadderEvidenceResolvesAndAgreesWithCorpus checks the cross-file
// invariants that have only ever been checked by hand: every
// evidence_row_ids entry resolves to a real corpus row, that row's own
// category_id/rung_number agrees with where the ladder cites it,
// row_count matches the actual evidence list length, and a rung's
// populated/empty status agrees with whether it actually carries
// evidence. Any of these silently drifting means a citation the
// reasoner builds either can't resolve at all, or resolves to the
// wrong practice -- exactly the failure this project's whole design
// exists to prevent.
func TestLadderEvidenceResolvesAndAgreesWithCorpus(t *testing.T) {
	ladder, err := LoadLadder()
	if err != nil {
		t.Fatalf("LoadLadder: %v", err)
	}
	corpus, _ := loadFullCorpus(t)

	for _, cat := range ladder.Categories {
		for _, rung := range cat.Rungs {
			populated := rung.Definition != nil
			if populated && len(rung.EvidenceRowIDs) == 0 {
				t.Errorf("%s rung %d has a definition but no evidence_row_ids", cat.CategoryID, rung.Rung)
			}
			if !populated && len(rung.EvidenceRowIDs) > 0 {
				t.Errorf("%s rung %d has no definition but %d evidence_row_ids", cat.CategoryID, rung.Rung, len(rung.EvidenceRowIDs))
			}
			if rung.RowCount != len(rung.EvidenceRowIDs) {
				t.Errorf("%s rung %d: row_count=%d but evidence_row_ids has %d entries", cat.CategoryID, rung.Rung, rung.RowCount, len(rung.EvidenceRowIDs))
			}
			for _, id := range rung.EvidenceRowIDs {
				row, ok := corpus[id]
				if !ok {
					t.Errorf("%s rung %d cites evidence row %q, which doesn't exist in corpus.jsonl", cat.CategoryID, rung.Rung, id)
					continue
				}
				if row.CategoryID != cat.CategoryID {
					t.Errorf("%s rung %d cites row %q, but that row's own category_id is %q", cat.CategoryID, rung.Rung, id, row.CategoryID)
				}
				if row.RungNumber == nil || *row.RungNumber != rung.Rung {
					got := "null"
					if row.RungNumber != nil {
						got = fmt.Sprintf("%d", *row.RungNumber)
					}
					t.Errorf("%s rung %d cites row %q, but that row's own rung_number is %s", cat.CategoryID, rung.Rung, id, got)
				}
			}
		}
	}
}

// TestParallelizationCautionRowExistsInRealCorpus guards the one
// corpus reference in reasoner.go that lives outside ladder.json's
// evidence_row_ids entirely. reasoner_test.go's own fixtures overwrite
// this same id with synthetic content for their own tests, so nothing
// else ever checks the constant against the real embedded corpus.
func TestParallelizationCautionRowExistsInRealCorpus(t *testing.T) {
	corpus, err := LoadCorpus()
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if _, ok := corpus[parallelizationCautionRowID]; !ok {
		t.Errorf("parallelizationCautionRowID %q does not exist in the real embedded corpus.jsonl", parallelizationCautionRowID)
	}
}
