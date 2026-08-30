// Package install registers couloir's hooks in a Claude Code settings
// file. Mirrors basanite's internal/install package: a settings file held
// as generic JSON so unrelated keys survive the round-trip, an existing
// registration retargeted in place rather than duplicated, and a backup
// written before every overwrite.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Hook is one registration: a subcommand, the event it runs on, and why.
type Hook struct {
	Event   string
	Sub     string
	Matcher string // tool pattern; only PreToolUse needs one
	Async   bool
	Why     string
}

// Hooks is every hook couloir registers.
var Hooks = []Hook{
	{Event: "PreToolUse", Sub: "observe", Matcher: "*",
		Why: "record trust_qa/agent_invocation tool-call facts into the calibration substrate"},
	{Event: "Stop", Sub: "transcript-scan", Async: true,
		Why: "scan the session transcript for context_mgmt/prompting_structure/model_routing facts"},
	{Event: "SessionStart", Sub: "transcript-scan", Async: true,
		Why: "catch up on transcript facts from prior sessions"},
	{Event: "UserPromptSubmit", Sub: "nudge",
		Why: "surface an optional next-rung/frontier suggestion when one is ripe"},
}

// Change describes one hook's outcome, for the report to the user.
type Change struct {
	Hook   Hook
	Action string // "added", "updated", "unchanged"
	Was    string // the previous command, when updated
}

// Settings is a Claude Code settings file held as generic JSON so that
// every key couloir does not know about survives the round-trip.
type Settings struct {
	raw map[string]any
}

// Load reads a settings file; a missing one is an empty settings object.
func Load(path string) (*Settings, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Settings{raw: map[string]any{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return &Settings{raw: m}, nil
}

// Apply registers every hook at bin, returning what changed. An existing
// couloir registration for the same subcommand is rewritten in place
// rather than duplicated, so re-running after a rebuild to a new path
// repoints the hooks instead of stacking a second copy.
func (s *Settings) Apply(bin string) []Change {
	hooks, _ := s.raw["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		s.raw["hooks"] = hooks
	}
	var changes []Change
	for _, h := range Hooks {
		want := bin + " " + h.Sub
		groups, _ := hooks[h.Event].([]any)
		if found, was, matcherChanged := retarget(groups, h.Sub, want, h.Matcher); found {
			action := "updated"
			if was == want && !matcherChanged {
				action = "unchanged"
			}
			changes = append(changes, Change{Hook: h, Action: action, Was: was})
			continue
		}
		entry := map[string]any{"type": "command", "command": want}
		if h.Async {
			entry["async"] = true
		}
		hooks[h.Event] = append(groups, map[string]any{
			"matcher": h.Matcher,
			"hooks":   []any{entry},
		})
		changes = append(changes, Change{Hook: h, Action: "added"})
	}
	return changes
}

// retarget points an existing couloir registration for sub at want, in
// place. It matches on the subcommand rather than the full path
// precisely so a hook left over from an older install location is
// repaired, not duplicated.
func retarget(groups []any, sub, want, matcher string) (found bool, was string, matcherChanged bool) {
	for _, g := range groups {
		gm, _ := g.(map[string]any)
		inner, _ := gm["hooks"].([]any)
		for _, e := range inner {
			em, _ := e.(map[string]any)
			cmd, _ := em["command"].(string)
			if !isCouloirSub(cmd, sub) {
				continue
			}
			em["command"] = want
			if existing, _ := gm["matcher"].(string); existing != matcher && len(inner) == 1 {
				gm["matcher"] = matcher
				matcherChanged = true
			}
			return true, cmd, matcherChanged
		}
	}
	return false, "", false
}

// isCouloirSub reports whether cmd invokes `couloir <sub>`, at any path.
func isCouloirSub(cmd, sub string) bool {
	f := strings.Fields(cmd)
	if len(f) < 2 || f[1] != sub {
		return false
	}
	return filepath.Base(f[0]) == "couloir"
}

// Remove strips every couloir registration, dropping groups left empty.
func (s *Settings) Remove() []Change {
	hooks, _ := s.raw["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}
	var changes []Change
	for _, h := range Hooks {
		groups, _ := hooks[h.Event].([]any)
		kept := make([]any, 0, len(groups))
		removed := false
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			inner, _ := gm["hooks"].([]any)
			keptInner := make([]any, 0, len(inner))
			for _, e := range inner {
				em, _ := e.(map[string]any)
				cmd, _ := em["command"].(string)
				if isCouloirSub(cmd, h.Sub) {
					removed = true
					continue
				}
				keptInner = append(keptInner, e)
			}
			if len(keptInner) == 0 {
				continue // the group held only our hook
			}
			gm["hooks"] = keptInner
			kept = append(kept, gm)
		}
		if len(kept) == 0 {
			delete(hooks, h.Event)
		} else {
			hooks[h.Event] = kept
		}
		action := "unchanged"
		if removed {
			action = "removed"
		}
		changes = append(changes, Change{Hook: h, Action: action})
	}
	if len(hooks) == 0 {
		delete(s.raw, "hooks")
	}
	return changes
}

// Registered reports the command currently registered for each hook.
func (s *Settings) Registered() map[string]string {
	out := map[string]string{}
	hooks, _ := s.raw["hooks"].(map[string]any)
	for _, h := range Hooks {
		groups, _ := hooks[h.Event].([]any)
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			inner, _ := gm["hooks"].([]any)
			for _, e := range inner {
				em, _ := e.(map[string]any)
				if cmd, _ := em["command"].(string); isCouloirSub(cmd, h.Sub) {
					out[h.Event] = cmd
				}
			}
		}
	}
	return out
}

// Bytes renders the settings as they will be written.
func (s *Settings) Bytes() ([]byte, error) {
	b, err := json.MarshalIndent(s.raw, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Save backs up the current file, then writes atomically via a temp file
// and a rename.
func (s *Settings) Save(path string) (backup string, err error) {
	b, err := s.Bytes()
	if err != nil {
		return "", err
	}
	if old, err := os.ReadFile(path); err == nil {
		backup = path + ".couloir-backup"
		if err := os.WriteFile(backup, old, 0o600); err != nil {
			return "", err
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return backup, err
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return backup, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return backup, err
	}
	if err := tmp.Close(); err != nil {
		return backup, err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return backup, err
	}
	return backup, os.Rename(tmp.Name(), path)
}

// Settled reports whether every hook is already where it should be.
func Settled(changes []Change) bool {
	for _, c := range changes {
		if c.Action != "unchanged" {
			return false
		}
	}
	return len(changes) > 0
}

// DefaultPath is the user-level Claude Code settings file.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// Render formats the changes for the terminal.
func Render(changes []Change, bin string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "couloir %s\n\n", bin)
	for _, c := range changes {
		fmt.Fprintf(&b, "  %-9s %-11s %s\n", c.Action, c.Hook.Event, c.Hook.Why)
		if c.Action == "updated" && c.Was != "" {
			fmt.Fprintf(&b, "  %-9s %-11s was: %s\n", "", "", c.Was)
		}
	}
	return b.String()
}

// RenderStatus formats what is registered right now.
func RenderStatus(reg map[string]string, path string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "hooks in %s\n\n", path)
	for _, h := range Hooks {
		mark, detail := "not registered", ""
		if cmd, ok := reg[h.Event]; ok {
			mark, detail = "registered", cmd
		}
		fmt.Fprintf(&b, "  %-11s %-15s %s\n", h.Event, mark, detail)
	}
	return b.String()
}
