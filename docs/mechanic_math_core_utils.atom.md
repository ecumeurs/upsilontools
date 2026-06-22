---
id: mechanic_math_core_utils
status: DRAFT
type: MECHANIC
layer: IMPLEMENTATION
priority: 3
tags: math,utility
parents:
  - [[shared:vision_upsilon_vision]]
dependents: []
human_name: Core Math Utilities
version: 1.0
---

# Core Math Utilities

## INTENT
Provide fundamental, reusable mathematical operations for integer and floating-point types to ensure consistency across the engine.

## THE RULE / LOGIC
- **Abs/AbsFloat:** Return the non-negative value of a given number.
- **Min/Max:** Return the smallest or largest of two given numbers.
- **LinearProgressionAt:** Calculate a value at a specific point `x` along a linear gradient between `(min_x, min_y)` and `(max_x, max_y)`.

## TECHNICAL INTERFACE
- **Code Tag:** `@spec-link [[mechanic_math_core_utils]]`
- **Test Names:** `TestMathCoreUtils`

## EXPECTATION
- Abs(-5) returns 5.
- Min(10, 20) returns 10.
- Max(10, 20) returns 20.
- LinearProgressionAt correctly interpolates values based on the provided range.
