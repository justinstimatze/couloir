// Command couloir is the Lens: a live PreToolUse hook that records raw
// trust_qa/agent_invocation facts into an append-only substrate, plus the
// install subcommand that wires the hook into Claude Code's settings.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/justinstimatze/couloir/internal/action"
	"github.com/justinstimatze/couloir/internal/calibration"
	"github.com/justinstimatze/couloir/internal/gate"
	"github.com/justinstimatze/couloir/internal/install"
	"github.com/justinstimatze/couloir/internal/lens"
	"github.com/justinstimatze/couloir/internal/reasoner"
	"github.com/justinstimatze/couloir/internal/substrate"
	"github.com/justinstimatze/couloir/internal/transcriptscan"
)

// version is overridden via -ldflags at release; see Makefile and the
// go-cli-versioning skill for why this isn't a hand-maintained const.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: couloir <observe|transcript-scan|nudge|install|uninstall|status|version>")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "observe":
		runObserve()
	case "transcript-scan":
		runTranscriptScan()
	case "nudge":
		runNudge()
	case "install":
		runInstall()
	case "uninstall":
		runUninstall()
	case "status":
		runStatus()
	case "version":
		fmt.Println(buildVersion())
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(1)
	}
}

func buildVersion() string {
	if version != "dev" {
		return version
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	var rev string
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}

var validSessionID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`).MatchString

// runObserve is the PreToolUse hook entrypoint. It fails open on every
// path: a lost observation beats a hook that fails loudly in front of
// every tool call.
func runObserve() {
	var in lens.PreToolUseInput
	if json.NewDecoder(os.Stdin).Decode(&in) != nil || !validSessionID(in.SessionID) {
		return
	}

	obsPath, err := substrate.ObservationsPath()
	if err != nil {
		return
	}
	cursorPath, err := lens.CursorPath(in.SessionID)
	if err != nil {
		return
	}

	cur := lens.LoadCursor(cursorPath)
	observations, newCur := lens.Classify(in, cur, time.Now(), genID)
	concObs, newCur := lens.CheckConcurrentSessions(in.SessionID, newCur, time.Now(), genID)
	if concObs != nil {
		observations = append(observations, *concObs)
	}

	for _, o := range observations {
		_ = substrate.Append(obsPath, o)
	}
	_ = lens.SaveCursor(cursorPath, newCur)
}

func genID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

type stopOrSessionStartInput struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
}

// runTranscriptScan is the Stop/SessionStart hook entrypoint — the
// second Lens mechanism, reading the session transcript incrementally
// from a per-session checkpoint. Fails open exactly like runObserve: any
// error at any step just returns, no output, no crash in front of a
// turn boundary.
func runTranscriptScan() {
	var in stopOrSessionStartInput
	if json.NewDecoder(os.Stdin).Decode(&in) != nil || !validSessionID(in.SessionID) || in.CWD == "" {
		return
	}

	lockPath, err := transcriptscan.LockPath(in.SessionID)
	if err != nil {
		return
	}
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return // another scan is already in progress for this session
	}
	lf.Close()
	defer os.Remove(lockPath)

	transcriptPath, err := transcriptscan.TranscriptPath(in.CWD, in.SessionID)
	if err != nil {
		return
	}
	f, err := os.Open(transcriptPath)
	if err != nil {
		return // no transcript yet
	}
	defer f.Close()

	cpPath, err := transcriptscan.CheckpointPath(in.SessionID)
	if err != nil {
		return
	}
	cp := transcriptscan.LoadCheckpoint(cpPath)

	info, err := f.Stat()
	if err != nil {
		return
	}
	if cp.ByteOffset > info.Size() {
		cp.ByteOffset = 0 // file shrank/rotated -- rescan from the start rather than error
	}
	if _, err := f.Seek(cp.ByteOffset, io.SeekStart); err != nil {
		return
	}

	obsPath, err := substrate.ObservationsPath()
	if err != nil {
		return
	}

	result := transcriptscan.Classify(f, cp.LastModel, in.SessionID, time.Now(), genID)
	for _, o := range result.Observations {
		_ = substrate.Append(obsPath, o)
	}

	cp.ByteOffset += result.BytesRead
	cp.LastModel = result.LastModel
	_ = transcriptscan.SaveCheckpoint(cpPath, cp)
}

// nudgeTailSize bounds how many recent observations Gate sees per turn
// — generous relative to Gate's own per-category windows (30 events /
// 3-5 sessions), not the whole file.
const nudgeTailSize = 1000

// runNudge is the UserPromptSubmit hook entrypoint: Gate -> Reasoner ->
// Action's cooldown-gated selection, printing additionalContext only
// when something was actually chosen. Silent, correct output on every
// other path -- no ripe signal is an expected v1 outcome, not a bug.
func runNudge() {
	_ = json.NewDecoder(os.Stdin).Decode(&struct{}{}) // drain stdin; no field here affects the logic below

	obsPath, err := substrate.ObservationsPath()
	if err != nil {
		return
	}
	tail, err := substrate.Tail(obsPath, nudgeTailSize)
	if err != nil {
		return
	}

	now := time.Now()
	estimates := gate.Estimate(tail, now)

	if calPath, err := calibration.Path(); err == nil {
		if statePath, err := calibration.PredictionsStatePath(); err == nil {
			calibration.RecordPredictions(calPath, statePath, estimates, now, genID)
		}
	}

	ladder, err := reasoner.LoadLadder()
	if err != nil {
		return
	}
	corpus, err := reasoner.LoadCorpus()
	if err != nil {
		return
	}
	suggestions := reasoner.Suggest(estimates, ladder, corpus)

	cooldownPath, err := action.CooldownPath()
	if err != nil {
		return
	}
	lockPath, err := action.CooldownLockPath()
	if err != nil {
		return
	}
	release, locked := action.AcquireCooldownLock(lockPath)
	if !locked {
		return // a concurrent session holds it; skip rather than risk clobbering its update
	}
	defer release()

	cooldown := action.LoadCooldown(cooldownPath)

	chosen, ok := action.Select(suggestions, cooldown, now, nudgeCooldownDuration())
	if !ok {
		return
	}

	cooldown.LastShown[chosen.CategoryID] = now
	_ = action.SaveCooldown(cooldownPath, cooldown)

	out, err := renderNudgeOutput(chosen)
	if err != nil {
		return
	}
	fmt.Println(out)
}

// nudgeCooldownDuration reads COULOIR_NUDGE_COOLDOWN_HOURS, falling
// back to action.DefaultCooldown for an unset or invalid value.
func nudgeCooldownDuration() time.Duration {
	v := os.Getenv("COULOIR_NUDGE_COOLDOWN_HOURS")
	if v == "" {
		return action.DefaultCooldown
	}
	hours, err := strconv.Atoi(v)
	if err != nil || hours <= 0 {
		return action.DefaultCooldown
	}
	return time.Duration(hours) * time.Hour
}

type hookOutput struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// renderNudgeOutput frames a suggestion exactly per README's framing:
// the next rung up, and separately a frontier example, both explicitly
// optional/fyi, never a ceiling comparison.
func renderNudgeOutput(s *reasoner.Suggestion) (string, error) {
	var b strings.Builder
	if s.CurrentRung > 0 {
		fmt.Fprintf(&b, "couloir (optional, fyi): your %s practice currently sits around rung %d.", s.CategoryID, s.CurrentRung)
	} else {
		fmt.Fprintf(&b, "couloir (optional, fyi): concurrent %s activity observed.", s.CategoryID)
	}
	if s.NextRung != nil {
		fmt.Fprintf(&b, " One rung up: %s Source: %s (%s).", s.NextRung.Definition, s.NextRung.SourceName, s.NextRung.SourceURL)
	}
	if s.Frontier != nil {
		fmt.Fprintf(&b, " For reference, a frontier example (not a bar to clear): %s Source: %s (%s).", s.Frontier.Definition, s.Frontier.SourceName, s.Frontier.SourceURL)
	}
	if s.Caution != nil {
		fmt.Fprintf(&b, " %s Source: %s (%s).", s.Caution.Definition, s.Caution.SourceName, s.Caution.SourceURL)
	}

	// Without an explicit relay instruction, this text lands in the
	// assistant's own context and nothing obliges it to reach the human
	// user -- confirmed live in session: the block fired, the assistant
	// noted that it fired, and never spoke the actual content. Action
	// already did the ripeness/cooldown gating before this function ever
	// runs, so the assistant's only remaining decision is how to phrase
	// this, already made ripe and worth surfacing by the pipeline itself.
	fmt.Fprint(&b, "\n\nThis cleared couloir's ripeness and cooldown gates, so it won't repeat again soon. Relay it to the user in your own words somewhere in this reply, don't just let it sit here unspoken.")

	out := hookOutput{HookSpecificOutput: hookSpecificOutput{
		HookEventName:     "UserPromptSubmit",
		AdditionalContext: b.String(),
	}}
	j, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func runInstall() {
	bin, err := resolvedExecutable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir install: %v\n", err)
		os.Exit(1)
	}
	path, err := install.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir install: %v\n", err)
		os.Exit(1)
	}
	s, err := install.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir install: %v\n", err)
		os.Exit(1)
	}
	changes := s.Apply(bin)
	fmt.Print(install.Render(changes, bin))
	if install.Settled(changes) {
		return
	}
	backup, err := s.Save(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir install: %v\n", err)
		os.Exit(1)
	}
	if backup != "" {
		fmt.Printf("\nbacked up previous settings to %s\n", backup)
	}
}

func runUninstall() {
	path, err := install.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir uninstall: %v\n", err)
		os.Exit(1)
	}
	s, err := install.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir uninstall: %v\n", err)
		os.Exit(1)
	}
	changes := s.Remove()
	fmt.Print(install.Render(changes, ""))
	backup, err := s.Save(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir uninstall: %v\n", err)
		os.Exit(1)
	}
	if backup != "" {
		fmt.Printf("\nbacked up previous settings to %s\n", backup)
	}
}

func runStatus() {
	path, err := install.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir status: %v\n", err)
		os.Exit(1)
	}
	s, err := install.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couloir status: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(install.RenderStatus(s.Registered(), path))
}

// resolvedExecutable is this binary's own absolute path, symlinks
// resolved, so the settings.json entry points at a real file couloir
// install won't need to re-resolve on every hook invocation.
func resolvedExecutable() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil {
		return bin, nil
	}
	return resolved, nil
}
