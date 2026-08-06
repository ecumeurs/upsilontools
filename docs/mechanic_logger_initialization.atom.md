---
id: mechanic_logger_initialization
status: DRAFT
version: 1.0
priority: 3
dependents: []
layer: IMPLEMENTATION
tags: [logging, initialization]
parents:
  - [[shared:req_tech_debt_backlog]]
human_name: "Logger Initialization Mechanic"
type: MECHANIC
---

# Logger Initialization Mechanic

## INTENT
Initialize global and local loggers with specific formatters and output targets using Logrus.

## THE RULE / LOGIC
- **InitConsole:** Sets TextFormatter and outputs to Stdout at Debug level.
- **InitFile:** Sets JSONFormatter and outputs to an appended file at Debug level; fatal on open error.
- **Sub-Loggers:** Creates new Logrus instances with the same logic as global inits but returning the instance.

## TECHNICAL INTERFACE
- **Code Tag:** `@spec-link [[mechanic_logger_initialization]]`
- **Functions:** `InitConsole`, `InitFile`, `InitSubLogger`, `InitSubFile`

## EXPECTATION
Logs are successfully written to stdout in text format for console init, and to specified files in JSON format for file init. Sub-loggers operate independently with their own outputs.
