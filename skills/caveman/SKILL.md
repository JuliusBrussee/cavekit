---
name: caveman
description: |
  Caveman encoding for SPEC.md and spec-adjacent writes. Loaded by /spec, /build,
  /check. Cuts tokens vs prose while staying precise. Triggers on any write
  to SPEC.md or when user says "caveman", "compress this", "be brief".
---

# caveman — spec encoding

Applies to SPEC.md writes, spec-referencing prose, backprop entries.
Does NOT apply to code, error strings, commit messages, PR descriptions.

## GRAMMAR

- Drop articles (a, an, the).
- Drop filler (just, really, basically, simply, actually).
- Drop aux verbs where fragment works (is, are, was, were, being).
- Drop pleasantries.
- No hedging (skip "might", "perhaps", "could be worth").
- Fragments fine.
- Short synonyms: fix > implement, big > extensive, run > execute.

## SYMBOLS

Prefer over words:

```
→   leads to / becomes / on <x>
∴   therefore / fix
∀   for all / every
∃   exists / some
!   must / required
?   may / optional / unknown
⊥   never / forbidden / nil
≠   not equal
∈   in
∉   not in
≤   at most
≥   at least
&   and
|   or
§   section reference
```

## PRESERVE VERBATIM

Never compress:

- Code blocks, snippets, one-liners with backticks.
- Paths: `src/auth/mw.go`.
- URLs.
- Identifiers: function names, variable names, env vars.
- Numbers and versions.
- Error message strings.
- SQL, regex, JSON, YAML.
- Quoted strings.

## SHAPES

**Invariant**:
```
V<n>: <subject> <relation> <condition>
V1: ∀ req → auth check before handler
V2: token expiry ≤ current_time → reject
```

**Bug row** (pipe table under §B):
```
id|date|cause|fix
B1|2026-04-20|token `<` not `≤`|V2
```

**Task row** (pipe table under §T):
```
id|status|task|cites
T3|x|add auth mw|V1,I.api
```
Status: `x` done, `~` wip, `.` todo. Escape literal `|` as `\|`.

**Interface**:
```
<kind>: <name> → <shape>
api: POST /x → 200 {id:string}
cmd: `foo bar <arg>` → stdout JSON
env: FOO_KEY ! set
```

## EXAMPLES

**Bad**:
> The system should ensure that every incoming request is properly authenticated before being forwarded to its corresponding handler function.

**Good**:
> V1: ∀ req → auth check before handler

**Bad**:
> We discovered that the token expiration check in the middleware was using a strict less-than comparison operator, which meant tokens were being rejected at the exact moment of their expiry.

**Good**:
> B1: token `<` not `≤` → reject @ expiry boundary.

**Bad**:
> The POST endpoint at /x accepts a JSON body and returns a 200 response with an object containing the created id.

**Good**:
> api: POST /x → 200 {id}

## BOUNDARIES

- User asks for prose explanation → switch to normal English.
- Spec documents for external review (RFC, pitch) → normal English.
- Commit message → normal English (git readers expect it).
- Diff comment in code → normal English.

## WHEN UNSURE

If cutting a word loses a fact, keep it. Caveman is compression, not amputation.
