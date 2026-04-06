# Cavekit — Example Conversations

## Greenfield: Task management API

```
user: /ck:sketch

claude: What are you building?

user: A task management API. Users create projects, add tasks with priorities
      and due dates, assign tasks to team members. REST API, PostgreSQL.

claude: Decomposing into domains...

      Created 4 kits:
      - cavekit-users.md (4 requirements, 12 acceptance criteria)
      - cavekit-projects.md (6 requirements, 18 acceptance criteria)
      - cavekit-tasks.md (9 requirements, 31 acceptance criteria)
      - cavekit-notifications.md (3 requirements, 8 acceptance criteria)

      Dependency order: users → projects → tasks → notifications
      Next: /ck:map

user: /ck:map

claude: Reading 4 kits, 22 requirements...

      Generated build site: 34 tasks, 5 tiers
      - Tier 0: DB schema, user model (3 tasks)
      - Tier 1: User CRUD, project model (5 tasks)
      - Tier 2: Project CRUD, task model (8 tasks)
      - Tier 3: Task CRUD, assignments (10 tasks)
      - Tier 4: Notifications (8 tasks)

      Next: /ck:make

user: /ck:make

claude: Loop activated! 34 tasks, 20 max iterations.

      --- Iteration 1 ---
      T-001: Database Schema Setup (Tier 0, cavekit-users.md R1)
      Build PASS, Tests PASS
      Committed: feat(users): add User model and migration

      --- Iteration 2 ---
      T-002: User Registration Endpoint (Tier 0, cavekit-users.md R1)
      ...

      --- Iteration 18 ---
      All tasks done. Build passes. Tests pass.
      <promise>CAVEKIT COMPLETE</promise>

      Done. 34 tasks, 18 iterations.
```

## Brownfield: Adding collaboration to existing Next.js app

```
user: /ck:sketch --from-code

claude: Exploring codebase... Next.js 14, Prisma, NextAuth.

      Found 6 domains. Created kits:
      - cavekit-auth.md (5 requirements — all satisfied by existing code)
      - cavekit-documents.md (7 requirements — all satisfied)
      - cavekit-editor.md (6 requirements — all satisfied)
      - cavekit-collaboration.md (3 requirements — all [GAP], not implemented)
      - cavekit-comments.md (4 requirements — all satisfied)
      - cavekit-notifications.md (3 requirements — 2 satisfied, 1 [GAP])

      4 requirements are gaps (not yet implemented).

user: I only want the collaboration features.
      /ck:map --filter collaboration

claude: Reading cavekit-collaboration.md (3 requirements)...

      Generated build site: 8 tasks, 3 tiers
      - Tier 0: WebSocket setup, presence types
      - Tier 1: Cursor presence, CRDT integration
      - Tier 2: Conflict resolution, live sync

user: /ck:make --filter collaboration

claude: Loop activated! 8 tasks.

      --- Iteration 1 ---
      T-001: WebSocket Server (Tier 0, R1)
      ...

      --- Iteration 8 ---
      All done. <promise>CAVEKIT COMPLETE</promise>

user: Now notifications with peer review.
      /ck:map --filter notifications
      /ck:make --filter notifications --peer-review

claude: Archived previous cycle → context/impl/archive/20260314-201500/
      Configured Codex (gpt-5.4) as peer reviewer.

      --- Iteration 1 (BUILD) ---
      T-001: Notification Service (R1)
      Committed.

      --- Iteration 2 (REVIEW) ---
      Calling Codex...
      Finding: HIGH — missing retry logic for email delivery
      Fixed. Committed.

      --- Iteration 6 (REVIEW) ---
      No new findings. All tasks done.
      <promise>CAVEKIT COMPLETE</promise>
```

## The flow

```
/ck:sketch      →  kits
/ck:map  →  build site
/ck:make      →  code
```
