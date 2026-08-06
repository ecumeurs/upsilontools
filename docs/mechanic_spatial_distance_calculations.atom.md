---
id: mechanic_spatial_distance_calculations
status: DRAFT
human_name: Spatial Distance Calculations
type: MECHANIC
tags: math,spatial,distance
parents:
  - [[shared:req_tech_debt_backlog]]
layer: IMPLEMENTATION
version: 1.0
priority: 3
dependents: []
---

# Spatial Distance Calculations

## INTENT
Provide standardized methods for calculating distances between points in 2D and 3D space using Manhattan distance.

## THE RULE / LOGIC
- **Distance (2D):** `Abs(x1-x2) + Abs(y1-y2)`
- **Distance3D (3D):** `Abs(x1-x2) + Abs(y1-y2) + Abs(z1-z2)`
- Supports both integer and `float64` coordinate types.

## TECHNICAL INTERFACE
- **Code Tag:** `@spec-link [[mechanic_spatial_distance_calculations]]`
- **Test Names:** `TestDistanceCalculations`

## EXPECTATION
- Distance(0,0, 3,4) returns 7 (Manhattan).
- Distance3D(0,0,0, 1,1,1) returns 3.
