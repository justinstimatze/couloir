# Changelog

## Unreleased

(none)

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
