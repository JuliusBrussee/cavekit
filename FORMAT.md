# SPEC.md FORMAT

Single file. Project root. Every cavekit command reads it.

## SECTIONS

Fixed order, fixed headers, addressable. A section may be absent (skip it —
e.g. §R only exists if `/research` ran) but is never reordered.

```
# SPEC

## §G GOAL
one line. what code must do.

## §C CONSTRAINTS
- bullet. non-negotiable boundary.
- bullet. tech/lang/lib locked in.

## §I INTERFACES
external surface. what world sees.
- cmd: `foo bar` → stdout JSON
- api: POST /x → 200 {id}
- file: `config.yaml` schema …
- env: `FOO_KEY` required

## §R RESEARCH
optional. external-knowledge log. pipe table. present only if `/research` ran.
durable ∴ build never re-derives & never hallucinates lib facts.
id|topic|finding|src
R1|jwt lib|`jose` > `jsonwebtoken` — maintained, ESM, 0 deps|github.com/panva/jose
R2|rate limit|token bucket ok @ our scale|<url>

## §V INVARIANTS
numbered. testable. each ! MUST hold.
V1: ∀ req → auth check before handler
V2: token expiry ≤ ⊥ allowed
V3: DB write ! in transaction

## §T TASKS
pipe table. ids monotonic (never reused). status: `x` done / `~` wip / `.` todo.
id|status|task|cites
T1|.|scaffold repo|-
T2|.|impl §I.api POST /x|V2
T3|x|add §V.1 middleware|V1,I.api

## §B BUGS
pipe table. backprop log. each row = bug + invariant that catches recurrence.
id|date|cause|fix
B1|2026-04-20|token `<` not `≤`|V2
B2|2026-04-21|race on write|V3
```

**Table cell rules**: literal `|` → escape as `\|`. Backticks OK. Cells trimmed. Empty = `-`.

## ADDRESSING

`§<S>.<n>` = section.item. `§V.2` = invariants section, item 2.
Commands, commits, PRs all reference by §. Zero ambiguity.

## CAVEMAN ENCODING

Default for every section. Rules:

- Drop articles (a, an, the). Drop filler.
- Drop aux verbs (is, are, was) where fragment works.
- Short synonyms (fix > implement).
- Fragments fine.

**Preserve verbatim**: code, paths, identifiers, URLs, numbers, error strings, SQL, regex.

**Symbols** (save tokens, machine-readable):

```
→   leads to / becomes / triggers
∴   therefore / fix
∀   for all / every
∃   exists / some
!   must
?   may / optional
⊥   never / impossible / forbidden
≠   not equal / differs from
∈   in / member of
∉   not in
≤   at most
≥   at least
&   and
|   or
```

**Bad** (v1 prose):

> The authentication middleware must verify the token expiry on every request before allowing the handler to execute.

**Good** (v2 caveman):

> V1: ∀ req → auth check before handler

**Bad** (prose bug note):

> Fixed a bug where token expiry comparison used strict less-than instead of less-than-or-equal, causing tokens to be rejected exactly at their expiry timestamp.

**Good** (v2 caveman):

> B1: token `<` not `<=` ∴ tokens rejected @ expiry. §V.2 now ! `≤`.

## WHY CAVEMAN FOR SPECS

Spec loaded every invocation. Fewer tokens = fewer dollars & faster reads.
Human skims fast too. Symbols unambiguous.

## ONE FILE RULE

Big project → more sections, not more files. grep ceremony kills agent speed.
If SPEC.md > 500 lines, compact §B (old bugs drop oldest) before splitting.

## WRITES — SECTIONED OWNERSHIP

Each verb owns specific sections. No verb rewrites a section it does not own.
That rule alone kills the "tool deleted my spec" failure mode.

| command | writes | section |
|---|---|---|
| `/grill` | sharpens | §G + §C (proposes; writes on OK) |
| `/spec new` | creates | all |
| `/spec amend` | edits | named only |
| `/spec bug` | appends | §B + §V |
| `/research` | appends | §R |
| `/review` | hardens | §V (+ risk report, no silent rewrite) |
| `/build` | flips | §T status cell `.` → `~` → `x` |
| `/deepen` | proposes | §I + §V + §T (writes on OK) |
| `/check` | — | read only |

`spec` is the general editor — any cross-cutting edit routes through it. Every
other verb shows a diff and touches only its own sections. Compaction (dropping
oldest §B rows when SPEC.md > 500 lines, per ONE FILE RULE) is the one sanctioned
§B rewrite — route it through `spec`, which shows the diff before dropping rows.

## RIGHT-SIZE

Ceremony scales to blast radius, never to ego. One-line fix → just `/build`.
New feature in a shared module → `/grill` then `/review` first. The full
grill→spec→research→review→build chain is for genuinely uncertain or
high-blast-radius work, ⊥ for a typo. Skip any verb that would cost more
attention than the change is worth.

That is whole format.
