// Command backtest is a throwaway, read-only analysis tool: it runs
// transcriptscan.Classify() against every historical Claude Code session
// transcript on this machine (never the live checkpoint files, never the
// live substrate) to generate real tuning data for Gate's ripeness
// constants in one pass, instead of waiting for couloir's own hooks to
// accumulate it organically. Not wired into couloir's own CLI dispatch
// or install — a one-off, run manually, not shipped.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/justinstimatze/couloir/internal/substrate"
	"github.com/justinstimatze/couloir/internal/transcriptscan"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: backtest <output.jsonl>")
		os.Exit(1)
	}
	outPath := os.Args[1]

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "home dir:", err)
		os.Exit(1)
	}
	projectsDir := filepath.Join(home, ".claude", "projects")

	projEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read projects dir:", err)
		os.Exit(1)
	}

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create output:", err)
		os.Exit(1)
	}
	defer out.Close()
	enc := json.NewEncoder(out)

	n := 0
	newID := func() string { n++; return fmt.Sprintf("bt%d", n) }
	now := time.Now()

	var allObs []substrate.Observation
	filesScanned, filesSkipped := 0, 0
	var bytesScanned int64
	sessionsWithSignal := map[string]bool{}

	for _, projEntry := range projEntries {
		if !projEntry.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, projEntry.Name())
		files, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
				continue
			}
			sessionID := f.Name()[:len(f.Name())-len(".jsonl")]
			fh, err := os.Open(filepath.Join(projDir, f.Name()))
			if err != nil {
				filesSkipped++
				continue
			}
			result := transcriptscan.Classify(fh, "", sessionID, now, newID)
			fh.Close()

			filesScanned++
			bytesScanned += result.BytesRead
			for _, o := range result.Observations {
				sessionsWithSignal[o.SessionID] = true
				allObs = append(allObs, o)
				_ = enc.Encode(o)
			}
		}
	}

	bySignal := map[string]int{}
	byCategory := map[string]int{}
	perSessionCount := map[string]map[string]int{} // signal -> session -> count
	for _, o := range allObs {
		bySignal[o.SignalType]++
		byCategory[o.CategoryID]++
		if perSessionCount[o.SignalType] == nil {
			perSessionCount[o.SignalType] = map[string]int{}
		}
		perSessionCount[o.SignalType][o.SessionID]++
	}

	fmt.Printf("scanned %d transcript files (%d skipped, unreadable), %.1f MB, %d sessions with >=1 signal, %d total observations\n\n",
		filesScanned, filesSkipped, float64(bytesScanned)/1e6, len(sessionsWithSignal), len(allObs))

	fmt.Println("by category_id:")
	printSortedCounts(byCategory)

	fmt.Println("\nby signal_type:")
	printSortedCounts(bySignal)

	fmt.Println("\nper-session distribution (min / median / max / sessions-with-any):")
	for _, sig := range sortedKeys(perSessionCount) {
		counts := valuesOf(perSessionCount[sig])
		sort.Ints(counts)
		fmt.Printf("  %-32s %4d / %4d / %4d / %d sessions\n",
			sig, counts[0], counts[len(counts)/2], counts[len(counts)-1], len(counts))
	}

	fmt.Printf("\nwrote %d raw observations to %s\n", len(allObs), outPath)
}

func printSortedCounts(m map[string]int) {
	for _, k := range sortedKeys(m) {
		fmt.Printf("  %-32s %d\n", k, m[k])
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func valuesOf(m map[string]int) []int {
	vals := make([]int, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}
	if len(vals) == 0 {
		vals = append(vals, 0)
	}
	return vals
}
