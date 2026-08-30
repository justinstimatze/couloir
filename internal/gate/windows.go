package gate

import "github.com/justinstimatze/couloir/internal/substrate"

// filterCategory returns observations for one category_id, preserving
// order (substrate.jsonl is append-only, so input order is chronological).
func filterCategory(obs []substrate.Observation, category string) []substrate.Observation {
	var out []substrate.Observation
	for _, o := range obs {
		if o.CategoryID == category {
			out = append(out, o)
		}
	}
	return out
}

// filterSignal returns observations matching one signal_type.
func filterSignal(obs []substrate.Observation, signal string) []substrate.Observation {
	var out []substrate.Observation
	for _, o := range obs {
		if o.SignalType == signal {
			out = append(out, o)
		}
	}
	return out
}

// ids collects observation ids, for citing evidence.
func ids(obs []substrate.Observation) []string {
	out := make([]string, 0, len(obs))
	for _, o := range obs {
		out = append(out, o.ID)
	}
	return out
}

// lastNByCount returns the most recent n observations (chronological
// input assumed) — the rolling-observation-count window shape.
func lastNByCount(obs []substrate.Observation, n int) []substrate.Observation {
	if len(obs) <= n {
		return obs
	}
	return obs[len(obs)-n:]
}

// lastNSessions returns every observation belonging to the n most
// recent distinct session_ids seen — the session-count window shape,
// for signals that are near-constant within a session (permission
// mode, model choice) rather than dense per-tool-call.
func lastNSessions(obs []substrate.Observation, n int) []substrate.Observation {
	seen := make(map[string]bool, n)
	keep := make(map[string]bool, n)
	for i := len(obs) - 1; i >= 0 && len(seen) < n; i-- {
		sid := obs[i].SessionID
		if !seen[sid] {
			seen[sid] = true
			keep[sid] = true
		}
	}
	var out []substrate.Observation
	for _, o := range obs {
		if keep[o.SessionID] {
			out = append(out, o)
		}
	}
	return out
}
