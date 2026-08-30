package action

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCooldownMissingFileIsEmpty(t *testing.T) {
	s := LoadCooldown(filepath.Join(t.TempDir(), "nope.json"))
	if len(s.LastShown) != 0 {
		t.Errorf("LoadCooldown on missing file = %+v, want empty", s.LastShown)
	}
}

func TestSaveLoadCooldownRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cooldown.json")
	now := time.Now().UTC().Truncate(time.Second)
	want := CooldownState{LastShown: map[string]time.Time{"trust_qa": now}}
	if err := SaveCooldown(path, want); err != nil {
		t.Fatalf("SaveCooldown: %v", err)
	}
	got := LoadCooldown(path)
	if !got.LastShown["trust_qa"].Equal(now) {
		t.Errorf("LastShown[trust_qa] = %v, want %v", got.LastShown["trust_qa"], now)
	}
}

func TestLoadCooldownDropsOnlyTheUnparseableEntry(t *testing.T) {
	// Reproduces the live 2026-08-30 incident: a hand-edited timestamp
	// missing the RFC3339 colon in its zone offset (+0000 instead of
	// +00:00) must not cost the other, well-formed categories their
	// real cooldown record.
	dir := t.TempDir()
	path := filepath.Join(dir, "cooldown.json")
	raw := `{"last_shown":{` +
		`"trust_qa":"2026-08-28T12:07:35+0000",` +
		`"context_mgmt":"2026-08-29T22:52:29-07:00",` +
		`"parallelization":"2026-08-30T11:44:11-07:00"` +
		`}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := LoadCooldown(path)
	if _, bad := got.LastShown["trust_qa"]; bad {
		t.Errorf("LastShown = %+v, want trust_qa dropped (unparseable offset)", got.LastShown)
	}
	if len(got.LastShown) != 2 {
		t.Errorf("LastShown = %+v, want the other 2 well-formed entries kept", got.LastShown)
	}
	if got.LastShown["context_mgmt"].IsZero() || got.LastShown["parallelization"].IsZero() {
		t.Errorf("LastShown = %+v, want context_mgmt and parallelization both parsed", got.LastShown)
	}
}

func TestAcquireCooldownLockExcludesAConcurrentHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cooldown.lock")
	release, ok := AcquireCooldownLock(path)
	if !ok {
		t.Fatal("first acquire failed, want success on an unheld lock")
	}
	defer release()

	if _, ok := os.Stat(path); ok != nil {
		t.Fatalf("lock file missing after acquire: %v", ok)
	}
	// A second acquire must not succeed while the first is held, and
	// must give up well inside the wait budget rather than hang.
	start := time.Now()
	_, second := AcquireCooldownLock(path)
	if second {
		t.Error("second acquire succeeded while the first still held the lock")
	}
	if elapsed := time.Since(start); elapsed > 2*cooldownLockWaitBudget {
		t.Errorf("second acquire took %v, want roughly the %v wait budget", elapsed, cooldownLockWaitBudget)
	}
}

func TestAcquireCooldownLockReleaseAllowsReacquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cooldown.lock")
	release, ok := AcquireCooldownLock(path)
	if !ok {
		t.Fatal("first acquire failed")
	}
	release()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("lock file still present after release: err=%v", err)
	}
	if _, ok := AcquireCooldownLock(path); !ok {
		t.Error("re-acquire after release failed, want success")
	}
}

func TestAcquireCooldownLockTakesOverAStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cooldown.lock")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	stale := time.Now().Add(-cooldownLockStaleAfter - time.Second)
	if err := os.Chtimes(path, stale, stale); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	release, ok := AcquireCooldownLock(path)
	if !ok {
		t.Fatal("acquire over a stale lock failed, want it taken over")
	}
	release()
}

func TestLoadCooldownWholeFileMalformedIsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cooldown.json")
	if err := os.WriteFile(path, []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := LoadCooldown(path)
	if len(got.LastShown) != 0 {
		t.Errorf("LastShown = %+v, want empty for a wholly malformed file", got.LastShown)
	}
}
