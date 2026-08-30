package substrate

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// StateDir resolves the couloir state directory ($XDG_STATE_HOME/couloir
// or ~/.local/state/couloir) and creates it.
func StateDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(base, "couloir")
	return dir, os.MkdirAll(dir, 0o755)
}

// ObservationsPath is the default substrate location.
func ObservationsPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "observations.jsonl"), nil
}

// Append writes one observation as a single JSON line. Concurrent sessions
// each append whole lines under a few hundred bytes, well under the
// kernel's atomic-write boundary for O_APPEND — no lock file needed for a
// personal single-writer-per-line tool.
func Append(path string, obs Observation) error {
	b, err := json.Marshal(obs)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(b)
	return err
}

// Tail reads the last n observations from an observations file. JSONL
// has no line index to seek by, so this still reads the whole file
// linearly, but keeps only the last n decoded rows in memory rather
// than the whole file's content — bounded memory for Gate's per-turn
// read, not a full-file load on every UserPromptSubmit call. A missing
// file returns an empty slice, not an error.
func Tail(path string, n int) ([]Observation, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]Observation, 0, n)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var o Observation
		if json.Unmarshal(sc.Bytes(), &o) != nil {
			continue
		}
		if len(buf) < n {
			buf = append(buf, o)
			continue
		}
		copy(buf, buf[1:])
		buf[n-1] = o
	}
	return buf, sc.Err()
}
