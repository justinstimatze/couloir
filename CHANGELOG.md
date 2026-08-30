# Changelog

## Unreleased

(none)

## [0.1.1] — 2026-08-30 — robustness pass

Every threshold and pattern list up to this point had only ever been
checked against one person's real transcripts (Go/JS-heavy). Broadens
`matchVerificationCommand` with test/lint/build patterns for Ruby,
PHP, .NET, Java/Kotlin, Elixir, and Swift, with tests pinning both the
newly-covered commands and the ones still deliberately unmatched (a
project-local wrapper script has no substring that's safe to match
without risking false positives). README's Status section now says
single-user usage plainly instead of "real usage."

Two checks that had only ever been reasoned about are now automated:
a stress test reproducing the original cooldown-lock incident under
real concurrent load (40 truly-simultaneous callers, race detector
on, 30 repeated runs) rather than the earlier sequential-only tests,
and a corpus/ladder cross-file integrity check riding the existing
`go test` CI step — every evidence citation resolves and agrees with
its source row, no duplicate corpus ids, every row carries real
provenance.

## [0.1.0] — 2026-08-30 — first public release

Core pipeline built and running end to end: `PreToolUse` and
`Stop`/`SessionStart` hooks record literal facts into an append-only
substrate, Gate infers a deterministic per-category rung estimate from
those facts, Reasoner looks up a cited next-rung and frontier example
from a sourced corpus, and Action surfaces at most one suggestion per
turn through `UserPromptSubmit`, gated by ripeness and a per-category
cooldown. No LLM call anywhere in that path. Six categories covered —
`trust_qa`, `agent_invocation`, `context_mgmt`, `prompting_structure`,
`model_routing`, `parallelization` — each backed by real, attributed
sourcing in `data/corpus.jsonl`.

Notable fixes since the initial build:

- The injected `additionalContext` now ends with an explicit
  instruction telling the assistant to relay it, rather than leaving a
  ripe, cooldown-cleared suggestion to reach the user only if the
  assistant happened to mention it.
- `action-cooldown.json`'s load-select-save sequence is now locked
  against concurrent sessions racing it — the file is shared across
  every session on a host, and an unlocked read-modify-write could
  silently drop one session's update.
- `LoadCooldown` drops only the one unparseable timestamp in the
  cooldown file instead of discarding every category's record when a
  single entry fails to parse.
