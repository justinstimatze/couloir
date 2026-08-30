package reasoner

import "testing"

func TestLoadLadderReadsEmbeddedFile(t *testing.T) {
	l, err := LoadLadder()
	if err != nil {
		t.Fatalf("LoadLadder: %v", err)
	}
	if len(l.Categories) != 6 {
		t.Errorf("got %d categories, want 6", len(l.Categories))
	}
	cat := l.CategoryByID("trust_qa")
	if cat == nil {
		t.Fatal("expected a trust_qa category")
	}
	if len(cat.Rungs) != 8 {
		t.Errorf("trust_qa has %d rungs, want 8", len(cat.Rungs))
	}
}

func TestLoadCorpusReadsEmbeddedFile(t *testing.T) {
	rows, err := LoadCorpus()
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if len(rows) < 100 {
		t.Errorf("got %d corpus rows, want at least 100", len(rows))
	}
}

func TestHighestPopulatedRungSkipsNullDefinitions(t *testing.T) {
	l, err := LoadLadder()
	if err != nil {
		t.Fatalf("LoadLadder: %v", err)
	}
	cat := l.CategoryByID("prompting_structure")
	if cat == nil {
		t.Fatal("expected a prompting_structure category")
	}
	frontier := cat.HighestPopulatedRung()
	if frontier == nil {
		t.Fatal("expected a populated frontier rung")
	}
	// Rungs 7-8 are documented no_corpus_evidence cells (definition:
	// null) -- the frontier must not be one of them.
	if frontier.Rung >= 7 {
		t.Errorf("HighestPopulatedRung() = rung %d, want < 7 (7-8 are null-definition cells)", frontier.Rung)
	}
}
