// Package reasoner is the one new package allowed to read
// data/ladder.json/data/corpus.jsonl at runtime — Lens and Gate never
// do, staying free of a stale-cache problem. Fully deterministic
// lookup, no LLM call: given Gate's rung estimate, cite the next rung
// up and, separately, the category's frontier, straight from the
// corpus's own evidence — never synthesized text.
package reasoner

import (
	"bufio"
	"encoding/json"

	"github.com/justinstimatze/couloir/data"
)

// Ladder mirrors data/ladder.json's schema.
type Ladder struct {
	SchemaVersion string     `json:"schema_version"`
	Categories    []Category `json:"categories"`
}

type Category struct {
	CategoryID string `json:"category_id"`
	Label      string `json:"label"`
	Rungs      []Rung `json:"rungs"`
}

type Rung struct {
	Rung           int      `json:"rung"`
	Definition     *string  `json:"definition"`
	RowCount       int      `json:"row_count"`
	Status         string   `json:"status"`
	EvidenceRowIDs []string `json:"evidence_row_ids"`
}

// CorpusRow mirrors the fields of data/corpus.jsonl this package cites.
type CorpusRow struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	Practice   string `json:"practice"`
	SourceName string `json:"source_name"`
	SourceURL  string `json:"source_url"`
	Excerpt    string `json:"excerpt"`
}

// LoadLadder reads the embedded ladder.
func LoadLadder() (*Ladder, error) {
	b, err := data.FS.ReadFile("ladder.json")
	if err != nil {
		return nil, err
	}
	var l Ladder
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// LoadCorpus reads the embedded corpus into an id-indexed map.
func LoadCorpus() (map[string]CorpusRow, error) {
	f, err := data.FS.Open("corpus.jsonl")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows := map[string]CorpusRow{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var r CorpusRow
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		rows[r.ID] = r
	}
	return rows, sc.Err()
}

// CategoryByID finds a category, or nil.
func (l *Ladder) CategoryByID(id string) *Category {
	for i := range l.Categories {
		if l.Categories[i].CategoryID == id {
			return &l.Categories[i]
		}
	}
	return nil
}

// RungByNumber finds a rung within a category, or nil.
func (c *Category) RungByNumber(n int) *Rung {
	for i := range c.Rungs {
		if c.Rungs[i].Rung == n {
			return &c.Rungs[i]
		}
	}
	return nil
}

// HighestPopulatedRung is the highest rung with a real (non-null)
// definition — the "frontier" for this category.
func (c *Category) HighestPopulatedRung() *Rung {
	var best *Rung
	for i := range c.Rungs {
		if c.Rungs[i].Definition != nil {
			best = &c.Rungs[i]
		}
	}
	return best
}
