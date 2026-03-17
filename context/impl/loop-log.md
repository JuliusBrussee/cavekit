### Iteration 1 — 2026-03-17
- **Task:** T-001 — Go module init, go.mod with dependencies
- **Tier:** 0
- **Status:** DONE
- **Files:** go.mod, go.sum, cmd/blueprint/main.go
- **Validation:** Build P, Tests N/A, Acceptance 4/4
- **Next:** T-008 — Command executor abstraction

### Iteration 2 — 2026-03-17
- **Task:** T-008 — Command executor abstraction (for testability)
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/exec/executor.go, internal/exec/mock.go, internal/exec/executor_test.go
- **Validation:** Build P, Tests 5/5 P, Acceptance 3/3
- **Next:** T-007 — Session instance model and status enum

### Iteration 3 — 2026-03-17
- **Task:** T-007 — Session instance model and status enum
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/session/instance.go, internal/session/instance_test.go
- **Validation:** Build P, Tests 6/6 P, Acceptance 4/4
- **Next:** T-002 — Tmux session create/kill/exists

### Iteration 4 — 2026-03-17
- **Task:** T-002 — Tmux session create/kill/exists
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/tmux/session.go, internal/tmux/session_test.go
- **Validation:** Build P, Tests 6/6 P, Acceptance 6/6
- **Next:** T-003 — Tmux pane content capture

### Iteration 5 — 2026-03-17
- **Task:** T-003 — Tmux pane content capture
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/tmux/capture.go, internal/tmux/capture_test.go
- **Validation:** Build P, Tests 9/9 P, Acceptance 3/3
- **Next:** T-004 — Git worktree create/detect/remove

### Iteration 6 — 2026-03-17
- **Task:** T-004 — Git worktree create/detect/remove
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/worktree/worktree.go, internal/worktree/worktree_test.go
- **Validation:** Build P, Tests 6/6 P, Acceptance 5/5
- **Next:** T-005 — Frontier file discovery and name derivation

### Iteration 7 — 2026-03-17
- **Task:** T-005 — Frontier file discovery and name derivation
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/frontier/discovery.go, internal/frontier/discovery_test.go
- **Validation:** Build P, Tests 4/4 P, Acceptance 5/5
- **Next:** T-006 — Frontier markdown parsing

### Iteration 8 — 2026-03-17
- **Task:** T-006 — Frontier markdown parsing (tasks, tiers, table rows)
- **Tier:** 0
- **Status:** DONE
- **Files:** internal/frontier/parser.go, internal/frontier/parser_test.go
- **Validation:** Build P, Tests 7/7 P, Acceptance 4/4
- **Next:** T-011 — Tmux input injection (Tier 1, unblocks T-020, T-022)

### Iteration 9 — 2026-03-17
- **Task:** T-011 — Tmux input injection (keystrokes, prompts)
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/tmux/input.go, internal/tmux/input_test.go
- **Validation:** Build P, Tests 4/4 P, Acceptance 3/3
- **Next:** T-010 — Tmux status detection

### Iteration 10 — 2026-03-17
- **Task:** T-010 — Tmux status detection (active/prompt via content hashing)
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/tmux/status.go, internal/tmux/status_test.go
- **Validation:** Build P, Tests 5/5 P, Acceptance 3/3
- **Next:** T-012 — Git diff stats

### Iteration 11 — 2026-03-17
- **Task:** T-012 — Git diff stats (files changed, insertions, deletions)
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/worktree/diff.go, internal/worktree/diff_test.go
- **Validation:** Build P, Tests 5/5 P, Acceptance 3/3
- **Next:** T-013 — Worktree discovery

### Iteration 12 — 2026-03-17
- **Task:** T-013 — Worktree discovery (scan sibling dirs)
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/worktree/discover.go, internal/worktree/discover_test.go
- **Validation:** Build P, Tests 2/2 P, Acceptance 3/3
- **Next:** T-014 — Task status tracking from impl files

### Iteration 13 — 2026-03-17
- **Task:** T-014 — Task status tracking from impl files
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/frontier/tracking.go, internal/frontier/tracking_test.go
- **Validation:** Build P, Tests 6/6 P, Acceptance 5/5
- **Next:** T-015 — Frontier status classification

### Iteration 14 — 2026-03-17
- **Task:** T-015 — Frontier status classification (done/in-progress/available)
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/frontier/status.go, internal/frontier/status_test.go
- **Validation:** Build P, Tests 5/5 P, Acceptance 4/4
- **Next:** T-017 — Progress summary string generation

### Iteration 15 — 2026-03-17
- **Task:** T-017 — Progress summary string generation
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/frontier/progress.go, internal/frontier/progress_test.go
- **Validation:** Build P, Tests 4/4 P, Acceptance 3/3
- **Next:** T-016 — Frontier multi-candidate ranking and selection

### Iteration 16 — 2026-03-17
- **Task:** T-016 — Frontier multi-candidate ranking and selection
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/frontier/ranking.go, internal/frontier/ranking_test.go
- **Validation:** Build P, Tests 6/6 P, Acceptance 5/5
- **Next:** T-018 — Bubbletea app shell (Tier 1, last remaining)

### Iteration 17 — 2026-03-17
- **Task:** T-019 — Lipgloss styles and constants + T-018 — Bubbletea app shell
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/tui/styles.go, internal/tui/app.go, internal/tui/app_test.go
- **Validation:** Build P, Tests 8/8 P, Acceptance T-019 3/3, T-018 5/5
- **Next:** T-009 — PTY-based tmux attach/detach (only remaining Tier 1 task)

### Iteration 18 — 2026-03-17
- **Task:** T-009 — PTY-based tmux attach/detach with Ctrl+Q
- **Tier:** 1
- **Status:** DONE
- **Files:** internal/tmux/attach.go, internal/tmux/terminal.go, internal/tmux/attach_test.go
- **Validation:** Build P, Tests 2/2 P, Acceptance 5/5
- **Note:** Limited unit testing due to PTY requirements; acceptance verified via code review of spec criteria
- **Next:** Tier 2 tasks — starting with highest-value unblocked tasks
