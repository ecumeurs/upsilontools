---
id: mechanic_mech_logger_initialization
status: DRAFT
version: 1.0
priority: 3
dependents: []
layer: IMPLEMENTATION
tags: [logging, initialization]
parents:
  - [[requirement_req_observability_logging]]
human_name: "Logger Initialization Mechanic"
type: MECHANIC
---

# New Atom

## INTENT
Initialize global and local loggers with specific formatters and output targets using Logrus.

## THE RULE / LOGIC
- **InitConsole:** Sets TextFormatter and outputs to Stdout at Debug level.
- **InitFile:** Sets JSONFormatter and outputs to an appended file at Debug level; fatal on open error.
- **Sub-Loggers:** Creates new Logrus instances with the same logic as global inits but returning the instance.

## TECHNICAL INTERFACE
- **Code Tag:** `@spec-link [[mech_logger_initialization]]`
- **Functions:** `InitConsole`, `InitFile`, `InitSubLogger`, `InitSubFile`

## EXPECTATION
Logs are successfully written to stdout in text format for console init, and to specified files in JSON format for file init. Sub-loggers operate independently with their own outputs.
