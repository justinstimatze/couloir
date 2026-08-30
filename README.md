# couloir

A personal Claude Code tool that watches how you actually work — which
tools you call, how you review a diff, when you switch models, how you
manage context — and calibrates roughly where you sit on an AI-coding
maturity ladder, one category at a time, entirely from observed
behavior. It never asks you to self-report, and it never collapses the
result to a single score: maturity is jagged, and a habit earned in one
category (say, verifying before trusting a diff) implies nothing about
another (say, running several agents at once).

When it has enough signal, it can surface one optional, cited nudge at
the start of a turn: the next rung up in a category you're ready for,
and — separately — a frontier example from someone further along, framed
as filling in an unknown unknown rather than a bar to clear. It stays
silent otherwise, and that silence is the expected, common outcome.

## Where this ladder comes from

The shape — several categories, each with its own graded rungs, no
single collapsed score — comes from Steve Yegge's individual-developer
AI-coding ladder, not the organization-scale maturity rubrics
(ai-levels.org, Gartner/McKinsey-style adoption whitepapers) that use
similar language: those measure a company's rollout, not a person's
actual habits. Only the shape carries over — every rung's content
comes from its own independently sourced, named citation.

## How it decides anything

Every claim couloir makes traces back to a real, attributed source. There
is no LLM call anywhere in the pipeline that runs during your session —
every rung estimate and every suggestion is deterministic, rule-based,
and file-backed:

```
Claude Code hooks           couloir observes literal facts only —
(PreToolUse, Stop,          never a maturity guess. That inference
SessionStart)          -->  step belongs to the next stage alone.
        |
        v
   substrate            an append-only, per-machine log of raw facts
   (observations.jsonl)  (permission mode, edit-then-verify sequences,
                          model switches, compaction events, ...)
        |
        v
     gate               deterministic aggregation: facts -> a rung
                         estimate per category, or an honest "not
                         enough signal yet" / "this fact doesn't map
                         onto the ladder" — never a forced guess
        |
        v
   reasoner              looks up the next populated rung (and,
                          separately, a frontier example) in a sourced
                          corpus, cites it by name and URL
        |
        v
    action                picks at most one suggestion per turn, only
                          for a category with real confidence and out
                          of its own cooldown, and injects it as
                          optional context on UserPromptSubmit
```

The corpus behind `reasoner` is built from real, attributed sources —
engineering blogs, tool documentation, interviews, primary essays —
never hand-written filler. A rung with no sourced evidence stays
genuinely empty rather than getting templated text, and every quoted
excerpt is short and attributed, never presented as more than it is.

## Install

Pure Go with zero external dependencies — `go.mod` names only the
standard library, so a fresh clone builds without fetching anything
else.

```
make build      # or: make install, to put couloir on your PATH
./couloir install
```

`couloir install` registers couloir's hooks into
`~/.claude/settings.json` (backing up the previous file first) and is
safe to re-run — it retargets an existing registration in place rather
than duplicating it. `couloir uninstall` removes them. `couloir status`
shows what's currently registered.

## Status

Early, and every threshold so far reflects one person's usage — the
author's own, hundreds of real sessions deep but a single developer's
habits and cadence, not yet checked against anyone else's. Several
rungs across the ladder have no reliable signal source yet and are
documented as such rather than approximated, and the thresholds that
decide when a suggestion is "ripe" are starting defaults, still
waiting on a second real user to measure them properly. Some of what
a rung measures — a team's
stated process, a UI-only setting, how carefully someone reviews
mid-stream — leaves no trace in a tool call or a session transcript at
all, and stays honestly unscored rather than guessed at.

## Naming

A couloir is a steep, defined mountain gully used as a climbing route,
conventionally graded by difficulty — a staged path up, not a single
waypoint. Matches what this tracks: a ladder, not a score.

## License

See [LICENSE](LICENSE).
