# Spec-as-Code: AI Agent Operating Spec

> Version 3.1 — Three Pillars Model

## Core Model

| #   | Problem                                  | Mechanism      | Approach                                             | Verification                                |
| --- | ---------------------------------------- | -------------- | ---------------------------------------------------- | ------------------------------------------- |
| 1   | **Wrong output** (requirement ambiguity) | Disambiguation | Agent generates Examples → Human approves            | Human can tell in 30 seconds                |
| 2   | **Missing constraints** (context loss)   | Context Memory | Scoped CONTEXT.md (2-4, only at decision boundaries) | Human writes, Agent reads only              |
| 3   | **Unstable output** (non-deterministic)  | Verification   | Tests (unit / property / e2e / smoke)                | Machine runs automatically, no human needed |

**Order is irreversible: 1 → 2 → 3.**

### Core Principle

```
Humans judge. Agents labor.
Humans never write code. Agents never decide requirements.
```

**RED LINE: Humans must review Examples at least once. Unreviewed Examples = common-mode failure.**

---

## Phase A: Project Decomposition (new project, one-time)

| Step | Who   | What                                                                       | Time     |
| ---- | ----- | -------------------------------------------------------------------------- | -------- |
| A1   | Human | Write a project description (prompt)                                       | 5-10 min |
| A2   | Agent | Analyze → decompose into features → one md per feature, pre-fill Context   | Auto     |
| A3   | Human | Review feature list (filenames + one-line descriptions only, not contents) | 1-2 min  |

Agent-generated feature file structure:

```
spec-records/
├── 01-user-model.md         # no dependencies
├── 02-auth.md               # depends on: user-model
├── 03-payment-gateway.md    # depends on: user-model
├── 04-order-flow.md         # depends on: user-model, payment-gateway
└── 05-notification.md       # depends on: order-flow
```

**Agent must annotate inter-feature dependencies.** Human proceeds through Phase B in dependency order.

---

## Phase B: Per-Feature Execution (repeat for each feature)

| Step | Who   | What                                                                                           | Time           |
| ---- | ----- | ---------------------------------------------------------------------------------------------- | -------------- |
| B1   | Human | Fill in Intent (one sentence)                                                                  | 30 sec         |
| B2   | Agent | Generate Examples (≥5: happy / edge / error)                                                   | Auto           |
| B3   | Human | Review Examples → approve / edit / supplement (**RED LINE**)                                   | 30 sec - 1 min |
| B4   | Agent | Generate tests + implementation + property tests → run tests → fix counterexamples (≤3 rounds) | Auto           |

```
If tests still fail after 3 rounds → Agent stops and reports → Human decides
```

### Why no "human reviews test code"?

```
Examples = human-readable test specification
Test code = mechanical translation of Examples

If Examples are correct, test code is correct.
Asking humans to read pytest/vitest code = asking humans to do Agent's job = waste of time.
The test framework IS the code review.
```

---

## Human Time Budget

```
New project setup (one-time):
├── Write project description:  5-10 min
└── Review feature list:        1-2 min

Per feature:
├── Fill Intent:                30 sec
└── Review Examples:            30 sec - 1 min

A 10-feature project ≈ ~20 minutes total human time
```

**Human is the approver. Agent is the executor. Approvers stamp, they don't write.**

---

## Feature Spec File Template

Each feature md uses the following structure (see `spec-records/spec-b.md` for the blank template, `spec-records/spec-a.md` for a filled example):

```markdown
# Spec: {feature-name}

## Step 1: Context (Agent pre-fills / Human may edit)
{Context derived from project description, including dependencies}

## Step 2: Intent (Human fills in)
{One sentence describing what this feature does}

## Step 3: Examples (Agent generates → Human approves)
| #   | Input | Output | Note       |
| --- | ----- | ------ | ---------- |
| 1   | ...   | ...    | happy path |
| 2   | ...   | ...    | edge case  |
| 3   | ...   | ...    | error case |

Approved by: {approver}

## Step 4: Tests + Implementation (Agent auto-completes)
<!-- Agent generates tests + implementation, no human needed -->
<!-- Tests must cover all approved examples + ≥1 property test -->
```

---

## Context Memory: Scoped CONTEXT.md

### Placement Rules

| Needs CONTEXT.md                              | Does NOT need                          |
| --------------------------------------------- | -------------------------------------- |
| Module has its own business rules             | Utility directories (utils/, helpers/) |
| Module has its own technical conventions      | Small directories with 1-2 files       |
| A new Agent would make wrong assumptions here | Rules are identical to parent          |

Typical project: **2-4 CONTEXT.md files**, not dozens.

### Rules

1. **Human writes, Agent reads only** — Agent must not modify CONTEXT.md
2. **Scoped** — Only placed where rules differ from the parent level
3. **Short and stable** — Each < 200 lines; longer means wrong granularity
4. **If it can be a type constraint, don't put it in Context** — "Amounts use int (cents)" should be a code constraint, not just documentation

---

## Verification: Test Layers

| Layer               | Purpose                              | Author                                        | When                   |
| ------------------- | ------------------------------------ | --------------------------------------------- | ---------------------- |
| Examples regression | Verify I/O pairs                     | Agent (auto-generated from approved examples) | After every generation |
| Unit tests          | Cover branches and boundaries        | Agent                                         | After every generation |
| Property tests      | Find counterexamples beyond examples | Agent                                         | After every generation |
| E2E / Smoke         | Verify complete user workflows       | Agent (human defines scenarios)               | Before PR / merge      |

### Test Frameworks

| Language   | Unit         | Property        | E2E                    |
| ---------- | ------------ | --------------- | ---------------------- |
| Python     | `pytest`     | `hypothesis`    | `pytest` + HTTP client |
| TypeScript | `vitest`     | `fast-check`    | Playwright / Cypress   |
| Go         | `go test`    | `go test -fuzz` | `go test` + `httptest` |
| Rust       | `cargo test` | `proptest`      | `assert_cmd`           |

---

## Prohibitions

| #   | Prohibited                                                | Reason                                               |
| --- | --------------------------------------------------------- | ---------------------------------------------------- |
| 1   | Agent starts implementation before human reviews Examples | Common-mode failure                                  |
| 2   | Agent modifies CONTEXT.md                                 | Errors become persistent, polluting all future tasks |
| 3   | Agent skips failing tests                                 | Hiding problems is worse than exposing them          |
| 4   | Agent reads files not declared in context                 | Prevents hallucinated dependencies                   |
| 5   | Starting implementation without Examples                  | No disambiguation = guessing requirements            |
| 6   | Continuing after 3 failed fix rounds                      | >3 rounds = misunderstanding; hand off to human      |

---

## Project Structure

```
project/
├── CONTEXT.md                    # Project-level context (human writes)
├── spec-as-code/
│   ├── spec-as-code.md           # This document (methodology, read once)
│   └── spec-records/             # Spec records for each feature
│       ├── spec-b.md             # Blank template (copy for new features)
│       ├── spec-a.md             # Filled example (calculate-discount)
│       ├── 01-user-model.md
│       ├── 02-auth.md
│       └── ...
└── src/
    ├── payments/
    │   └── CONTEXT.md            # Module-level context (if needed)
    └── ...
```

---

## Relationship with AGENTS.md

| File                        | Responsibility                                                      |
| --------------------------- | ------------------------------------------------------------------- |
| AGENTS.md                   | Project-level Agent behavior rules (test coverage, code style, CI)  |
| CONTEXT.md                  | Module-level business rules and technical conventions               |
| spec-as-code.md (this file) | Methodology: how to drive verifiable code generation from Examples  |
| spec-records/*.md           | Per-feature spec instances (approval records + generated artifacts) |

---

*Humans judge. Agents labor. The test framework is the code review.*
