# Changelog

## Unreleased

Two new meta rows closing a gap surfaced while auditing frontier-scale
claims against independent verification: `t1` (`trust_qa`, rung_number
null) cites a peer-review-track study of 20,574 real coding-agent
sessions — 91.49% of visible misalignment episodes still needed
explicit user correction, and inaccurate self-reporting grows in share
even as the aggregate rate improves, a direct caution against leaning
on model self-introspection (`l1`) without independent verification
alongside it. `t2` (`parallelization`, rung_number null) cites Software
Improvement Group's independent ISO-25010 audit of Cursor's FastRender
swarm build — 1.3/5 maintainability, bottom 5% of systems SIG has
measured — as a caution against reading scale/throughput numbers alone
as evidence of production-grade output. A parallel check for frontier
labs' own internal orchestration scale (beyond what's already cited at
`i3`/`h5`/`h6`) came up empty; noted as a real gap, not filled with a
weak source.

Ran a house style-lint pass (`cope-gate`) against `data/corpus.jsonl` for
the first time — 17 violations, all pre-existing except one in `s1`.
Rewrote 13: the not-A-but-B flip pattern (7 rows), one reflexive
"load-bearing" intensifier, one "worth noting" hedge, and 4 balanced
two-beat clause constructions — wording only, no source, citation, or
rung content changed. The remaining 4 are JSONL-scanning artifacts (the
tool reads the file as flat prose and occasionally matches across a
record boundary, or applies a closing-summary heuristic to a data file
that has no closing summary) rather than real issues in any row's text.

Two new graded rows extending `trust_qa` rung 8 (`s1`) and `context_mgmt`
rung 7 (`s2`), sourced from Gas Town's own documentation (Steve Yegge's
multi-agent orchestrator, `gastownhall/gastown`) rather than paraphrased
from a secondary account — this corpus already cited Gas Town once
(`j6`, a failure case: a model upgrade breaking Yegge's own
orchestrator), but nothing about how the system actually works.
`s1` names Refinery, a merge-queue processor that bisects a failing
batch to isolate which change broke it before anything reaches main —
distinct from this rung's existing review/testing rows, this is
merge-time structural infrastructure. `s2` names Seance, on-demand
querying of a specific predecessor session's own reasoning — distinct
from this rung's existing ambient-context rows, this is retroactive
recall rather than forward-looking infrastructure. Both held to
confidence 0.6, below this corpus's vendor-documentation rows, given
single-source status and the operational failure already on record for
this same system.

Two new meta corpus rows (`r1`, `r2`, `rung_number: null`), paraphrased
from Addy Osmani's *Agentic Engineering* (O'Reilly, early-release
manuscript) — not quoted, per its prerelease status. `r1` corroborates
an existing meta claim that `prompting_structure`'s empty rungs 7-8
reflect a deliberate design conclusion — the frontier practice already
lives under other categories, and this row was never meant to fill
them. `r2` names the
autonomy/orchestration axis split as a caution alongside
`parallelization`'s existing cost caution. README's ladder-lineage
section now credits the same source for the two-axis critique that
couloir's separate-categories design already answers.

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
