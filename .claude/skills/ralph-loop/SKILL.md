---
name: ralph-loop
description: Autonomous task-by-task execution loop from an SDD spec file (Stop hook-based iteration)
argument-hint: "<spec-file-path>"
user-invocable: true
---

# /ralph-loop <spec-file>

Executes tasks from a spec file autonomously, one task per iteration. Uses the Stop hook (exit code 2) to continue in the same session after each task.

## Example

```text
/ralph-loop .specs/user-audit-log.md
```

## Mechanism

The loop uses the Stop hook pattern:

1. You execute ONE task from the spec
2. When you finish (try to stop), the `ralph-loop.sh` Stop hook fires
3. If tasks remain: hook returns exit 2 (continue) — you receive a stderr message with progress
4. If all tasks done: hook returns exit 0 — `stop-validate.sh` runs final validation
5. Each iteration adds context to the same session — be focused and concise

## Startup

1. Read the spec file path from argument
2. Validate the spec exists and has status `APPROVED` or `IN_PROGRESS`
3. Verify no other ralph-loop is active (no other `.active.md` files in `.specs/`)
4. Set status to `IN_PROGRESS` if not already
5. Create state file: `.specs/<name>.active.md` containing the spec file path (this signals the Stop hook)
6. Identify the next uncompleted `- [ ] TASK-N:` entry

## Per-Iteration Execution

**CRITICAL: Execute exactly ONE task per iteration. Do not batch tasks.**

For each task:

1. **Read the spec file** to find the current task (first unchecked `- [ ] TASK-N:`)
2. **Read relevant code** referenced in the spec's Design section
3. **Execute the task** as described — follow project conventions and existing patterns
4. **Verify**: run `go build ./...` to confirm compilation passes
5. **Mark complete**: change `- [ ] TASK-N:` to `- [x] TASK-N:` in the spec file
6. **Log**: append to the Execution Log section:

```markdown
### Iteration N — TASK-N (YYYY-MM-DD HH:MM)

<1-2 sentence summary of what was done and which files were created/modified>
```

7. **Stop** — let the hook decide whether to continue or finish

## On Final Task

After marking the last task complete:

1. The Stop hook detects all tasks done, returns exit 0
2. `stop-validate.sh` runs full validation (build + lint + tests)
3. If validation passes: set spec status to `DONE`
4. If validation fails: fix issues (stop-validate retries up to 3 times)
5. Suggest: "Run `/spec-review .specs/<name>.md` for a formal review against requirements"

## Resume After Interruption

If a loop was interrupted (Ctrl+C, crash, etc.):

1. The `.active.md` state file remains on disk
2. Running `/ralph-loop .specs/<name>.md` again picks up from the first unchecked task
3. No work is lost — completed tasks are already marked `[x]`

## Rules

- **ONE task per iteration** — never try to do multiple tasks in one go
- **Read the spec file first** every iteration — it is the single source of truth
- Never modify the spec's Requirements or Design sections during execution
- If a task is unclear or blocked, mark it `BLOCKED` in the spec, remove the `.active.md` file, and stop
- Follow project conventions: unique error variable names, existing patterns, etc.
- Be concise in responses — context accumulates across iterations
- For features with many tasks (15+), consider splitting into smaller specs

## Emergency Stop

To stop the loop at any time:

```bash
rm .specs/*.active.md
```

This removes the state file. The Stop hook will see no active loop and pass through normally.
