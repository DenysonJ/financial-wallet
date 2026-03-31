---
name: spec
description: Create a structured SDD specification (requirements, design, tasks) for a new feature or change
argument-hint: "<feature-description>"
user-invocable: true
---

# /spec <feature-description>

Creates a structured specification document following Specification-Driven Development (SDD) principles.

## Example

```text
/spec "Add audit logging to all user write operations"
```

## Workflow

### 1. Understand the Request

- Parse the feature description
- Identify affected domain(s) and code areas
- Determine the type of change: new feature, refactor, bug fix, new domain, etc.

### 2. Gather Context

- Read existing code for affected areas
- Check ADRs in `docs/adr/` for relevant architectural decisions
- Identify existing patterns to follow (use `user` and `role` domains as reference)
- Respect the project's chosen architecture (separate layers, collapsed, or hybrid)

### 3. Generate Spec

- Create `.specs/<feature-name>.md` from the template at `.specs/TEMPLATE.md`
- Fill in all sections: Context, Requirements, Design, Tasks, Validation Criteria
- Requirements should use **GIVEN/WHEN/THEN** format for unambiguous acceptance criteria
- Mark uncertain items with `[NEEDS CLARIFICATION]` instead of assuming
- Tasks must be:
  - Concrete and independently verifiable (`go build ./...` should pass after each)
  - Ordered logically for the feature (not necessarily by architecture layer)
  - Small enough to complete in a single focused iteration
  - Self-contained — each task description should be understandable without reading previous tasks

### 4. Present for Approval

- Display the spec to the user
- Set status to `DRAFT`
- Ask: "Review this spec. Edit anything you want, then approve to begin implementation."
- On approval, set status to `APPROVED`

## Rules

- Spec files go in `.specs/` directory
- File naming: lowercase, hyphen-separated: `.specs/user-audit-log.md`
- Never include tasks that require user decisions — ask upfront during spec creation
- Reference existing patterns: if a task is similar to existing code, note which files to use as reference
- Match spec depth to task complexity — a simple bug fix needs fewer sections than a new domain
- Architecture is flexible: the template recommends Clean Architecture but does not impose it. Adapt the spec to the project's chosen structure

## Integration

- After approval, run `/ralph-loop .specs/<name>.md` for autonomous execution
- Or execute tasks manually one at a time
- Use `/spec-review .specs/<name>.md` after implementation to verify against requirements
