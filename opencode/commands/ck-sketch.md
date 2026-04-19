---
description: Design-first drafting of kits for the OpenCode Cavekit portable workflow
argument-hint: [--from-code | REFS_PATH]
---

You are running **/ck-sketch** for the OpenCode Cavekit portable port.

Hard rules:
- Do not claim upstream runtime behavior.
- Do not write kit files until you have presented the proposed domain structure and the user has explicitly approved it.
- Keep kits implementation-agnostic: describe **what** must be true, never **how** to code it.
- Do not use subagent APIs or deep-research fan-out. Work directly with normal OpenCode tools.

Interpret `$ARGUMENTS`:
- `--from-code` → brownfield drafting/refinement from the current repo
- any other non-empty value → treat as a reference-material hint to inspect first
- empty → interactive drafting mode

## Step 1: Inspect current state before asking questions
Use Read/Glob/Grep and, when useful, Bash for `git log --oneline -10` to inspect:
- `README.md`
- `AGENTS.md`
- existing `context/kits/`
- existing `context/refs/`
- major source directories
- recent git history
- `DESIGN.md` if present

## Step 2: Frame current situation
- If kits already exist, explain whether you are refining, extending, or replacing them.
- If no kits exist, explain that you will propose a domain decomposition first.
- If the request spans multiple unrelated systems, propose a decomposition before drafting files.

## Step 3: Design conversation
- Ask 1-2 scoped questions at a time.
- Prefer multiple choice when possible.
- Focus on scope, users, constraints, success criteria, and boundaries.
- If existing kits already cover most domains, focus on drift, missing domains, or changed priorities.

## Step 4: Approval gate
Before writing any file, present:
- proposed kit/domain list
- brief scope of each kit
- notable cross-domain dependencies

Ask for explicit approval.

## Step 5: After approval, write kits
Ensure `context/kits/` exists, then create or update:
- `context/kits/cavekit-overview.md`
- `context/kits/cavekit-{domain}.md`

Each kit file should contain:
- YAML frontmatter with `created` and `last_edited`
- short scope statement
- R-numbered requirements
- testable acceptance criteria with unchecked checkboxes
- out-of-scope section
- cross-references when domains interact

Each requirement should be concrete enough that `/ck-map` can turn it into tasks.

## Step 6: Report
Return:
- domains touched
- files created/updated
- estimated requirement count
- exact next step: usually `/ck-map`

Be collaborative before approval. Precise after approval.
