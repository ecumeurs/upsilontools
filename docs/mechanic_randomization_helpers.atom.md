---
id: mechanic_randomization_helpers
status: DRAFT
priority: 3
dependents: []
human_name: Randomization Helpers
type: MECHANIC
version: 1.0
tags: random,utility
parents:
  - [[shared:req_tech_debt_backlog]]
layer: IMPLEMENTATION
---

# Randomization Helpers

## INTENT
Provide a centralized interface for both deterministic (seeded) and non-deterministic random number generation.

## THE RULE / LOGIC
- **Global Seeding:** `Seed()` uses the current system time. `SeedWith(s)` uses a specific 64-bit integer.
- **RandomInt(min, max):** Returns a random integer in the range `[min, max)`.
- **IntRange:** A struct representing a numeric range with a `Random()` method.
- **TesterRand:** Allows injecting a mock RNG function for deterministic testing.

## TECHNICAL INTERFACE
- **Code Tag:** `@spec-link [[mechanic_randomization_helpers]]`
- **Test Names:** `TestRandomizationHelpers`

## EXPECTATION
- RandomInt(10, 10) returns 10.
- RandomInt(0, 10) returns a value between 0 and 9.
- SeedWith ensures the same sequence of numbers is generated if called with the same seed.
