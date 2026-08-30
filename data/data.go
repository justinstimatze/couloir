// Package data embeds the corpus and ladder at build time. Reasoner
// needs them at runtime, but the nudge/transcript-scan hooks run from
// whatever project directory the user is actually working in — not from
// inside this repo's checkout — so a relative-path os.ReadFile would
// break as soon as the hook fires anywhere else. Embedding makes the
// binary self-contained instead.
package data

import "embed"

//go:embed ladder.json corpus.jsonl
var FS embed.FS
