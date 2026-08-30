package transcriptscan

// contextLimits is a best-effort table of per-model context-window
// sizes, used only to turn a summed token count into a rough
// utilization percentage. Not authoritative — Claude Code's own
// transcript carries no window-percentage field (confirmed by direct
// inspection), so this is the closest available proxy, not a fact.
var contextLimits = map[string]int{
	"claude-sonnet-5":   200_000,
	"claude-opus-5":     200_000,
	"claude-fable-5":    200_000,
	"claude-haiku-4-5":  200_000,
	"claude-sonnet-4-5": 200_000,
	"claude-opus-4-1":   200_000,
	"claude-opus-4":     200_000,
}

// defaultContextLimit is used for any model not in the table above,
// rather than refusing to compute a utilization sample at all.
const defaultContextLimit = 200_000

// contextLimitFor returns the best-known context window size for a
// model id, falling back to defaultContextLimit for an unrecognized
// or versioned/dated model string.
func contextLimitFor(modelID string) int {
	if n, ok := contextLimits[modelID]; ok {
		return n
	}
	return defaultContextLimit
}
