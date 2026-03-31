---
applies-to: ".specs/**"
---

# SDD Spec Rules

## Spec File Integrity

- Never modify the Requirements section during execution (only during DRAFT status)
- Never remove tasks — mark them as `[x]` (done) or `BLOCKED`
- Always append to Execution Log, never overwrite previous entries
- Status transitions: DRAFT -> APPROVED -> IN_PROGRESS -> DONE | FAILED

## Task Execution

- Each task must be independently verifiable (`go build ./...` should pass after each task)
- Tasks are architecture-agnostic — no mandatory layer ordering
- Order tasks logically for the feature, respecting the project's chosen structure
- If a task is unclear, mark it `BLOCKED` with a reason and stop execution

## Naming

- Spec files: lowercase, hyphen-separated: `user-audit-log.md`, `role-permissions.md`
- Active state files: `<name>.active.md` (auto-created by ralph-loop, do not edit manually)
