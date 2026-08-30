package action

import (
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/reasoner"
)

func TestSelectNothingWhenNoSuggestions(t *testing.T) {
	_, ok := Select(nil, CooldownState{LastShown: map[string]time.Time{}}, time.Now(), DefaultCooldown)
	if ok {
		t.Error("expected ok=false with no suggestions")
	}
}

func TestSelectPrefersNeverShown(t *testing.T) {
	now := time.Now()
	suggestions := []reasoner.Suggestion{
		{CategoryID: "trust_qa", Confidence: "high"},
		{CategoryID: "context_mgmt", Confidence: "high"},
	}
	cooldown := CooldownState{LastShown: map[string]time.Time{
		"trust_qa": now.Add(-72 * time.Hour), // shown before, now out of cooldown
	}}
	got, ok := Select(suggestions, cooldown, now, DefaultCooldown)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.CategoryID != "context_mgmt" {
		t.Errorf("CategoryID = %q, want context_mgmt (never shown beats out-of-cooldown)", got.CategoryID)
	}
}

func TestSelectSkipsCategoryStillInCooldown(t *testing.T) {
	now := time.Now()
	suggestions := []reasoner.Suggestion{
		{CategoryID: "trust_qa", Confidence: "high"},
	}
	cooldown := CooldownState{LastShown: map[string]time.Time{
		"trust_qa": now.Add(-1 * time.Hour), // well within the 48h default
	}}
	_, ok := Select(suggestions, cooldown, now, DefaultCooldown)
	if ok {
		t.Error("expected ok=false while still in cooldown")
	}
}

func TestSelectLeastRecentlyShownWinsAmongEligible(t *testing.T) {
	now := time.Now()
	suggestions := []reasoner.Suggestion{
		{CategoryID: "trust_qa", Confidence: "high"},
		{CategoryID: "context_mgmt", Confidence: "high"},
	}
	cooldown := CooldownState{LastShown: map[string]time.Time{
		"trust_qa":     now.Add(-72 * time.Hour),
		"context_mgmt": now.Add(-100 * time.Hour),
	}}
	got, ok := Select(suggestions, cooldown, now, DefaultCooldown)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.CategoryID != "context_mgmt" {
		t.Errorf("CategoryID = %q, want context_mgmt (shown longer ago)", got.CategoryID)
	}
}

func TestSelectExcludesLowConfidence(t *testing.T) {
	now := time.Now()
	suggestions := []reasoner.Suggestion{
		{CategoryID: "trust_qa", Confidence: "low"},
	}
	_, ok := Select(suggestions, CooldownState{LastShown: map[string]time.Time{}}, now, DefaultCooldown)
	if ok {
		t.Error("expected ok=false for a low-confidence-only suggestion set")
	}
}
