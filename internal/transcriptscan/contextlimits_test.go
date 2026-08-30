package transcriptscan

import "testing"

func TestContextLimitForKnownModel(t *testing.T) {
	if got := contextLimitFor("claude-sonnet-5"); got != 200_000 {
		t.Errorf("contextLimitFor(claude-sonnet-5) = %d, want 200000", got)
	}
}

func TestContextLimitForUnknownModelFallsBack(t *testing.T) {
	if got := contextLimitFor("some-future-model-v9"); got != defaultContextLimit {
		t.Errorf("contextLimitFor(unknown) = %d, want %d", got, defaultContextLimit)
	}
}
