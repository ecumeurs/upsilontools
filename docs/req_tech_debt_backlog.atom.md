---
id: req_tech_debt_backlog
human_name: Technical Debt Backlog
type: REQUIREMENT
layer: BUSINESS
version: 1.0
status: STABLE
priority: 3
tags: [tech-debt, escape-hatch, governance]
parents: []
dependents: []
---
# Technical Debt Backlog

## INTENT
Acts as a temporary anchor for implementation atoms created without formal business requirements.

## THE RULE / LOGIC
Any ARCHITECTURE or IMPLEMENTATION atom that cannot yet be traced to a real business requirement must declare `[[req_tech_debt_backlog]]` as its parent rather than being committed without traceability. These atoms must be groomed and re-parented to proper business atoms during scheduled tech-debt cycles.

- **This is not a free pass.** Using this anchor must be intentional and visible to reviewers.
- **Cycle responsibility:** The team must periodically query `parents: [[req_tech_debt_backlog]]` and groom those atoms toward real requirements.
- **Acceptable scenarios:** rapid prototyping, emergency fixes, infrastructure atoms with no direct user story.

## TECHNICAL INTERFACE (The Bridge)
- **Code Tag:** `@spec-link [[req_tech_debt_backlog]]` — do not use in source code; this is documentation-only.
- **Query:** `atd query --field parents --search req_tech_debt_backlog` to list all tech debt atoms.

## EXPECTATION (For Testing)
- No atom at ARCHITECTURE or IMPLEMENTATION layer is committed with an empty `parents` list.
- The count of atoms pointing here decreases over time, not increases.
