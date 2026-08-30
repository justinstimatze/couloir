// Package calibration is the predict+verdict log every hybrid-loops project
// starts on day one, per the skill's own guidance, even before a Gate or
// Reasoner exists to produce real predictions. Two kinds of row: a
// rung_prediction (Gate's eventual job — stays empty until Gate ships) and
// an observation_grounding (usable today — does a recorded substrate
// observation match what a human replaying the transcript agrees happened).
package calibration

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/justinstimatze/couloir/internal/substrate"
)

const SchemaVersion = "v0"

const (
	KindRungPrediction       = "rung_prediction"
	KindObservationGrounding = "observation_grounding"
)

// Prediction is a Gate rung guess, as of a point in time. Nil until Gate ships.
type Prediction struct {
	Rung       int     `json:"rung"`
	Confidence float64 `json:"confidence"`
	AsOf       string  `json:"as_of"`
}

// Verdict is a checked-against-ground-truth outcome for one entry.
type Verdict struct {
	Correct   bool   `json:"correct"`
	CheckedBy string `json:"checked_by"`
	CheckedAt string `json:"checked_at"`
	Note      string `json:"note,omitempty"`
}

// Entry is one calibration-log row.
type Entry struct {
	SchemaVersion        string      `json:"schema_version"`
	ID                   string      `json:"id"`
	LoggedAt             string      `json:"logged_at"`
	Kind                 string      `json:"kind"`
	SessionID            string      `json:"session_id,omitempty"`
	SubjectObservationID string      `json:"subject_observation_id,omitempty"`
	CategoryID           string      `json:"category_id,omitempty"`
	Predict              *Prediction `json:"predict"`
	Verdict              *Verdict    `json:"verdict"`
	Notes                string      `json:"notes,omitempty"`
}

// Path is the default calibration log location, alongside the substrate.
func Path() (string, error) {
	dir, err := substrate.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "calibration.jsonl"), nil
}

// Append writes one calibration entry as a single JSON line.
func Append(path string, e Entry) error {
	b, err := json.Marshal(e)
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
