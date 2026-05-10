---
trigger: always_on
---

# IDE Agent Ruleset: Atomic Traceable Documentation (ATD)

**Core Mandate:** You are operating in a codebase governed by Atomic Traceable Documentation (ATD). Documentation and code are not separate entities; they co-evolve as a verifiable graph. You must maintain bidirectional traceability from requirements to code to tests.

### 1. The Atom Blueprint
Every atom (`.atom.md`) is a single-responsibility file with strict YAML frontmatter and four mandatory H2 sections. You must adhere to this exact structure when conceptualizing or updating atoms:

```markdown
---
id: unique_slug
human_name: "Human Readable Name"
type: MECHANIC
layer: IMPLEMENTATION
version: 1.0
status: DRAFT
priority: 3
tags: [tag1, tag2]
parents:
  - [[parent_atom_id]]
dependents:
  - [[child_atom_id]]
---

# Human Readable Name

## INTENT
[One sentence: Why does this exist? No "and" or "also".]

## THE RULE / LOGIC
[The core specification. Use pseudo-code, formulas, or strict bullet points.]

## TECHNICAL INTERFACE (The Bridge)
- **API Endpoint:** `POST /v1/example` (if applicable)
- **Code Tag:** `@spec-link [[unique_slug]]`
- **Related Issue:** `#123`
- **Test Names:** `TestMyLogic1`, `TestMyLogic2`

## EXPECTATION (For Testing)
[Verifiable acceptance criteria for pass/fail testing. What must be true?]
```

### 2. Document Types & Bloat Factor Quick Reference

Use this table to determine the correct `type`, `layer`, and expected granularity when creating atoms. The **Bloat Factor** column shows the default `bloating_factor` from `.atd` config (1.0 = strictest, 0.1 = most relaxed).

| Type | Family | Typical Layer | Bloat Factor | Granularity |
|---|---|---|---|---|
| `REQUIREMENT` | Requirements | BUSINESS | 0.3 | High-level external contract |
| `RULE` | Logic | BUSINESS / ARCH | 0.8 | Single business constraint |
| `USER_STORY` | Requirements | BUSINESS | 0.1 | User-facing workflow |
| `API` | Interface | ARCHITECTURE | 0.1 | One contract, include payloads |
| `UI` | Interface | ARCHITECTURE | 0.8 | One screen or flow |
| `ENTITY` | Architectural | ARCHITECTURE | 0.8 | Single data model |
| `MECHANIC` | Logic | IMPLEMENTATION | 0.8 | One algorithm or validation |
| `MODULE` | Architectural | ARCHITECTURE | 0.3 | Broad grouping / Service |
| `DOMAIN` | Logic | BUSINESS | 0.8 | Narrative-driven context |

> **Rule:** Before creating an atom, check the bloat factor for its type. High factor (≥0.7) = laser-focused on ONE rule. Low factor (≤0.3) = broader scope is acceptable.

### 3. The "Minimum Atomic Scale" Rule
* Each atom file must describe exactly ONE state-changing rule.
* If an `## INTENT` statement requires the words "and" or "also", you must split the logic into multiple atoms.
* **Always check tolerances:** Use the table above or the `atd_config` tool with `bloating_factor` to retrieve the tolerance for a specific atom type.

### 4. File Modification & Tool Guardrails
* **Never rewrite an entire `.atom.md` file.** Always use the `atd_update` tool to surgically modify specific frontmatter fields or H2 sections.
* **Prioritize deterministic tools:** Use `atd_query`, `atd_trace`, `atd_crawl`, `atd_weave`, and `atd_update` for fast, token-free structural operations.
* **Delegate LLM tasks:** When semantic analysis, complex extraction, or auditing is required, do not do the analysis yourself. Instead, use the MCP's LLM-backed tools to offload the work to ATD's configured models and save your own context window:
  - `atd_discover` — three modes: find matching atoms for undocumented code (default), confirm a specific match (`atom` param), or propose a new atom skeleton (`new: true`)
  - `atd_recon` — shorthand confirm mode: validate whether a candidate file implements a specific atom
  - `atd_check` — unified coverage report: impl links (`@spec-link`) and test links (`@test-link`) in one pass; add `semantic: true` for LLM compliance check per link
  - `atd_search`, `atd_audit`, `atd_dissect` — semantic search, atom quality audit (bloat + collision), document decomposition
  - `atd_trace(summary=true)` — **MANDATORY** for getting narrative vertical context before code changes

### 5. The Day-to-Day Workflow
When asked to build a feature, fix a bug, or update code, you must follow this lifecycle loop:
* **Plan:** Use `atd_query` or `atd_search` to find existing relevant atoms. Create new `DRAFT` atoms using `atd_update` to capture new requirements before writing code.
* **Specify:** Ensure every new atom links upward using the `parents` field in the frontmatter. Run `atd_weave` to establish the downward dependency graph (`dependents`).
* **Implement:** Before writing any code, you MUST run `atd_trace(atom=..., summary=true)` to get a narrative assessment of the atom's context and impact. Then write the code. You must annotate the source code with `@spec-link [[atom_id]]` to map it to the implementation. Annotate tests with `@test-link [[atom_id]]`.
* **Verify:** Run `atd_check` to get a unified coverage report (impl links + test links) for the atoms touched by your changes. Then run `atd_trace` for the full health snapshot of each atom. Ensure implementation and test coverage metrics meet the required standards. Always ensure that a new atom has a link toward the upper layers (Business ← Architecture ← Implementation). If none are present that fits the need, raise the issue to the user. 
* **Evolve:** Before modifying any `STABLE` atom, you must run `atd_crawl` to assess the blast radius and impact on the rest of the system.

### 6. Surgical Traceability (Tag Placement)
* **No Global Headers:** Do not place `@spec-link` tags at the top of a source file unless the atom literally represents the entire architectural pattern of that file.
* **Target Logic Boundaries:** Place `@spec-link` tags directly above the specific class definition, function, decorator, or logical block that implements the atom.
* **Test Logic Boundaries:** Place `@test-link` tags directly above the specific function, decorator, or logical block that test the atom.
* **Discovery:** If you are unsure where to place tags in undocumented code, use `atd_discover` to get placement recommendations. If you already have a specific atom in mind, use `atd_recon` (or `atd_discover` with `atom` param) to confirm the match before tagging. If no matching atom exists yet, use `atd_discover` with `new: true` to get a proposed atom skeleton.

### 7. Respect the Documentation Hierarchy
* **BUSINESS Layer (`REQUIREMENT`, `USER_STORY`, `RULE`, etc.):** Treat these as low-volatility. Do not alter `STABLE` business atoms without explicit human permission. **Requirement:** When requesting this permission from the user, you must proactively run `atd_crawl` and present the impact analysis/blast radius to them.
* **ARCHITECTURE Layer (`MODULE`, `API`, `UI`, `ENTITY`):** Treat these as moderate-volatility. Always run an impact analysis (`atd_crawl`) before changing.
* **IMPLEMENTATION Layer (`MECHANIC`, etc.):** Treat these as high-volatility. Update these freely as you refactor or write new code.

### 8. Pragmatic Traceability & Health (The Trace Rule)
When using `atd_trace`, treat the resulting health metrics as a guide rather than a strict blocker. Apply the following logic:

**Top-Down Design is Expected:** It is perfectly acceptable for BUSINESS and ARCHITECTURE layer atoms to have a 0% implementation_rate or test_coverage_rate. Missing code/tests at this stage simply mean the feature is "on the to-do list." Do not stubbornly attempt to generate tests or code unless the user explicitly asks you to build the implementation.

**Implementations Require Roots:** The only strict warning you must act upon is missing ancestry. If you are creating or modifying an IMPLEMENTATION atom and atd_trace reports has_customer_origin: false, you must stop and ask the user for clarification. Code should not exist without a reason. Let the user define the missing upstream requirement before you proceed.

### 9. Workspace & Multi-Project Workflow
When a `.atd.workspace` file is present, you are in a multi-project environment. Each project has its own `.atd` config and `docs/` folder, but all are queryable through one workspace index.

**Agent Responsibilities:**
1. Call `atd_workspace_list` at the start of any workspace task to see available projects.
2. Determine the correct project from file paths, `@spec-link` tags, or explicit user direction. When uncertain, ask.
3. Switch context explicitly with `atd_workspace_use(project=...)` before any ATD operations.
4. Switch again whenever the task moves to a different project.

**Determining the Active Project:**
| Signal | Action |
|---|---|
| Task references a specific service folder | Use the matching project name |
| You see `@spec-link [[some_atom_id]]` | Find which project owns that atom |
| User says "in the API project" | Follow it literally |
| Ambiguous | Ask before proceeding |

**Cross-Project Atom References:**
When an atom in one project depends on an atom from another, use the `project:` prefix:
```markdown
parents:
  - [[upsilonapi:api_auth_login]]
```

**DO NOT:**
- Assume you're on the correct project without checking.
- Create atoms in the wrong project's `docs/` folder.
- Ignore workspace context in monorepo environments.

### 10. Common Patterns & Best Practices

**DO:**
- Start every feature with ATD: create or update atoms before writing any code.
- Use `atd_search` to find related atoms before starting new work — avoid creating duplicates.
- Place `@spec-link` tags directly above the specific function or block that implements the atom.
- Run `atd_weave` after creating new atoms to establish the downward dependency graph.
- Update atom `status` progressively: `DRAFT` → `REVIEW` → `STABLE`.
- When modifying a `STABLE` atom, always run `atd_crawl` first to assess blast radius (structural) and `atd_trace(summary=true)` for vertical context (semantic).

**DON'T:**
- Use file-level `@spec-link` tags unless the atom represents the entire file's architectural pattern.
- Create overly broad atoms — "and" or "also" in INTENT means you must split.
- Ignore atom `status`: implement `DRAFT` atoms only after reviewing their intent.
- Break existing `@spec-link` chains when refactoring — update tags, never silently delete them.
- Write code without an upstream atom. Missing `has_customer_origin` is a blocker — ask the user.

### Quick Reference

```bash
# Find atoms
atd_query(field="type", search="MECHANIC")
atd_search(query="turn timer implementation", scope="all")

# Create / update
atd_update(file="docs/new.atom.md", set=["id=new", "type=RULE", "layer=BUSINESS", "status=DRAFT"])
atd_weave()                          # rebuild dependency graph

# Traceability
atd_check()                         # impl + test link coverage report
atd_check(semantic=true)            # + LLM compliance check per link
atd_trace(atom="your_atom_id")       # full health snapshot
atd_trace(atom="your_atom_id", summary=true) # narrative contextual summary

# Impact analysis
atd_crawl()                          # blast radius before modifying STABLE atoms
atd_lint()                           # broken links, circular deps

# Workspace
atd_workspace_list()                 # list projects
atd_workspace_use(project="name")    # activate a project

# Discovery & mapping
atd_discover(file="src/foo.go")                  # find candidate atoms for undocumented code
atd_discover(file="src/foo.go", atom="rule_foo") # confirm a specific match
atd_discover(file="src/foo.go", new=true)        # propose a new atom skeleton
atd_recon(file="src/foo.go", atom="rule_foo")    # shorthand confirm
```