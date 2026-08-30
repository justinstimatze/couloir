package action

import (
	"time"

	"github.com/justinstimatze/couloir/internal/reasoner"
)

// DefaultCooldown is how long a category's nudge stays suppressed after
// being shown. A default, not a measured value -- adjustable via
// COULOIR_NUDGE_COOLDOWN_HOURS at the call site.
const DefaultCooldown = 48 * time.Hour

// Select picks at most one suggestion to surface this turn: among
// suggestions with medium or high confidence (Reasoner currently only
// ever produces "low" from... nothing — no caller sets it today, so this
// filter is defensive, not reachable by the current Suggest() output)
// that are out of cooldown, the one least recently shown wins
// (never-shown categories sort first). Returns ok=false when nothing
// qualifies -- Action never forces content.
func Select(suggestions []reasoner.Suggestion, cooldown CooldownState, now time.Time, cooldownDuration time.Duration) (best *reasoner.Suggestion, ok bool) {
	var bestPriority time.Time // zero value sorts before any real timestamp

	for i := range suggestions {
		s := &suggestions[i]
		if s.Confidence == "low" {
			continue
		}
		last, shown := cooldown.LastShown[s.CategoryID]
		if shown && now.Sub(last) < cooldownDuration {
			continue
		}
		if best == nil || last.Before(bestPriority) {
			best, bestPriority = s, last
		}
	}
	return best, best != nil
}
