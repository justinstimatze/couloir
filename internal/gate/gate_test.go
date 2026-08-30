package gate

import (
	"testing"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
)

func TestEstimateReturnsOnePerCategory(t *testing.T) {
	got := Estimate(nil, time.Now())
	if len(got) != 6 {
		t.Fatalf("Estimate(nil) returned %d estimates, want 6", len(got))
	}
	want := map[string]bool{
		substrate.CategoryTrustQA:            true,
		substrate.CategoryAgentInvocation:    true,
		substrate.CategoryContextMgmt:        true,
		substrate.CategoryPromptingStructure: true,
		substrate.CategoryModelRouting:       true,
		substrate.CategoryParallelization:    true,
	}
	for _, e := range got {
		if !want[e.CategoryID] {
			t.Errorf("unexpected category %q in Estimate() output", e.CategoryID)
		}
		delete(want, e.CategoryID)
		if e.State != StateInsufficientSignal {
			t.Errorf("category %q with no observations = %v, want insufficient_signal", e.CategoryID, e.State)
		}
	}
	if len(want) != 0 {
		t.Errorf("missing categories in Estimate() output: %+v", want)
	}
}
