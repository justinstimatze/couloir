package calibration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/justinstimatze/couloir/internal/gate"
	"github.com/justinstimatze/couloir/internal/substrate"
)

// confidenceValue maps Gate's string confidence onto the log's float64
// field. Coarse on purpose -- Gate itself only ever reasons in three
// buckets, so a finer scale here would be manufactured precision.
var confidenceValue = map[string]float64{"high": 0.9, "medium": 0.6, "low": 0.3}

// lastLogged tracks, per category, the last StateFloor rung actually
// written to the calibration log -- so RecordPredictions appends only on
// a real change, not once per UserPromptSubmit call. Gate recomputes
// fresh every turn; most turns report the same rung as the one before.
type lastLogged struct {
	Rung map[string]int `json:"rung"`
}

// PredictionsStatePath is the dedup-state location, alongside the
// substrate.
func PredictionsStatePath() (string, error) {
	dir, err := substrate.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "calibration-last-logged.json"), nil
}

func loadLastLogged(path string) lastLogged {
	b, err := os.ReadFile(path)
	if err != nil {
		return lastLogged{Rung: map[string]int{}}
	}
	var s lastLogged
	if json.Unmarshal(b, &s) != nil || s.Rung == nil {
		return lastLogged{Rung: map[string]int{}}
	}
	return s
}

func saveLastLogged(path string, s lastLogged) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".calibration-last-logged-*.tmp")
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

// RecordPredictions appends one rung_prediction entry per category whose
// Gate estimate is a new StateFloor rung -- different from the last one
// actually logged for that category, or the first ever seen for it.
// Non-floor estimates (Banded/Unmapped/InsufficientSignal) carry no
// single rung to log against and are skipped, same as Reasoner's own
// floor-only suggestion rule. Best-effort and silent: a lost calibration
// row is not worth failing a hook call over, same discipline as every
// other Append call in this project.
func RecordPredictions(logPath, statePath string, estimates []gate.RungEstimate, now time.Time, newID func() string) {
	state := loadLastLogged(statePath)
	changed := false
	for _, est := range estimates {
		if est.State != gate.StateFloor {
			continue
		}
		if prev, ok := state.Rung[est.CategoryID]; ok && prev == est.Rung {
			continue
		}
		entry := Entry{
			SchemaVersion: SchemaVersion,
			ID:            newID(),
			LoggedAt:      now.UTC().Format(time.RFC3339),
			Kind:          KindRungPrediction,
			CategoryID:    est.CategoryID,
			Predict: &Prediction{
				Rung:       est.Rung,
				Confidence: confidenceValue[est.Confidence],
				AsOf:       est.AsOf.UTC().Format(time.RFC3339),
			},
		}
		_ = Append(logPath, entry)
		state.Rung[est.CategoryID] = est.Rung
		changed = true
	}
	if changed {
		_ = saveLastLogged(statePath, state)
	}
}
