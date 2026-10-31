## Autonomous Execution & Engineering Governance

### 1. Approved SPEC authorizes full execution

A SPEC explicitly approved by the project lead constitutes authorization to fully execute the corresponding development cycle.

The agent must autonomously carry out all stages outlined in the `/sdd-workflow` without requiring human approval for each phase transition.

### 2. Mandatory Workflow

The `/sdd-workflow` remains mandatory and serves as the source of truth for:

- Stages and sequencing.
- Discovery and specification.
- Design and architectural decisions.
- TDD (Test-Driven Development).
- Implementation.
- Verification.
- Independent reviews.
- Completion criteria.

The agent may not bypass, simplify, or replace any steps of the workflow on its own initiative.

### 3. Operational Autonomy

Within the scope of an approved SPEC, the agent must:

- Execute all authorized phases.
- Continuously review its own work.
- Identify inconsistencies, risks, and gaps.
- Implement, test, correct, and repeat cycles.
- Address feedback from reviewers.
- Re-run verifications following corrections.
- Produce tangible evidence of compliance.
- Complete the workflow without relying on intermediate human intervention.

Do not request authorization for routine technical decisions already covered by the SPEC, design, architecture, or established standards.

### 4. Continuous Review and Quality

The agent is responsible for continuously verifying:

- Compliance with the SPEC.
- Compliance with the approved design.
- Adherence to the existing architecture.
- Quality, simplicity, and maintainability.
- Security and error handling. - Coverage of requirements.
- Absence of regressions.
- Evidence of acceptance criteria.

A phase is not considered complete simply because commands were executed successfully.

### 5. Limits of Autonomy

The agent must halt execution and request a human decision only when:

- There is a significant ambiguity or contradiction that cannot be resolved using approved artifacts.
- A change to a requirement or an approved architectural decision is necessary.
- There is a need to expand the authorized scope.
- There is a significant risk to security, integrity, or external impact.
- An external dependency prevents further progress.
- A persistent failure cannot be resolved safely.

### 6. Change Governance

- Never relax acceptance criteria to declare success.
- Never fabricate or omit evidence.
- Never modify components outside the authorized scope.
- Never initiate another epic or SPEC on one's own initiative.
- Adhere to commit and push gates, including LIM-03.
- Never interpret implementation autonomy as automatic authorization to publish changes.

### 7. Completion

The agent must carry the workflow through to the full completion of the approved SPEC.

Upon completion, provide a consolidated report containing:

- Deliverables completed.
- Files modified.
- Technical decisions.
- Tests and verifications.
- Verified acceptance criteria.
- Results of an independent review.
- Pending items and risks.

Stop upon the completion of the SPEC and await authorization before beginning a new scope.

## Guidelines for Updates

1. Read the current `AGENTS.md` before making any modifications.
2. Avoid duplicating existing rules.
3. Integrate new guidelines in a manner consistent with the current structure.
4. Retain all existing rules that do not conflict with the new ones. 5. If there is a conflict with any existing instruction, document it; do not resolve it by silently changing important rules.
5. Do not modify the implementation of `/sdd-workflow` or any other skills.
6. Provide the full diff and explain the changes.

Do not commit or push.
