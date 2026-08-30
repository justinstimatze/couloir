package transcriptscan

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/justinstimatze/couloir/internal/substrate"
)

// Checkpoint is how far this session's transcript has already been
// scanned, so a repeated Stop/SessionStart trigger reads only the new
// tail instead of rescanning a file that grows unbounded (3.8MB/2041
// lines observed for one real session).
type Checkpoint struct {
	ByteOffset int64  `json:"byte_offset"`
	LastModel  string `json:"last_model,omitempty"`
}

// CheckpointDir is where per-session transcript-scan checkpoints live.
func CheckpointDir() (string, error) {
	dir, err := substrate.StateDir()
	if err != nil {
		return "", err
	}
	cpDir := filepath.Join(dir, "transcript-checkpoint")
	return cpDir, os.MkdirAll(cpDir, 0o755)
}

// CheckpointPath is one session's checkpoint file.
func CheckpointPath(sessionID string) (string, error) {
	dir, err := CheckpointDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionID+".json"), nil
}

// LockPath is a per-session single-flight lock, so a Stop and a
// SessionStart trigger firing close together don't both scan the same
// new byte range and append duplicate observations.
func LockPath(sessionID string) (string, error) {
	dir, err := CheckpointDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionID+".lock"), nil
}

// LoadCheckpoint reads a session's checkpoint. Missing or corrupt is a
// zero-value checkpoint (scan from the start), never an error.
func LoadCheckpoint(path string) Checkpoint {
	b, err := os.ReadFile(path)
	if err != nil {
		return Checkpoint{}
	}
	var c Checkpoint
	if json.Unmarshal(b, &c) != nil {
		return Checkpoint{}
	}
	return c
}

// SaveCheckpoint writes a session's checkpoint atomically.
func SaveCheckpoint(path string, c Checkpoint) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
