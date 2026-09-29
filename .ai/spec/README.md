# Alerts Adapter - Specifications

A stateless Go component that polls OpenShift AlertManager for firing alerts and creates `AgenticRun` CRs (`agentic.openshift.io/v1alpha1`) to trigger automated remediation via the Lightspeed Agentic operator. Single-replica, create-only, poll-based design.

## Structure

| Layer | Path | Purpose |
|---|---|---|
| **what/** | `.ai/spec/what/` | System contracts. What the adapter must do: behavioral rules, configuration surface, deduplication logic, multicluster support. |
| **how/** | `.ai/spec/how/` | Codebase map. How the code is organized: module map, key abstractions, entry points. |

## Scope

Covers the alerts adapter's internal behavior. Cross-repo integration contracts (multicluster credential flow, hub-spoke interaction model) live in the parent spec at `ols/.ai/spec/what/alerts-adapter-multicluster.md`.

## Audience

AI agents. Content is optimized for precision and machine consumption.

## Quick Start

| Task | Start here |
|---|---|
| Understand the system | `what/system-overview.md` |
| Understand the reconcile loop | `what/poll-loop.md` |
| Understand alert-to-CR translation | `what/agenticrun-building.md` |
| Understand AlertManager interaction | `what/alert-retrieval.md` |
| Understand runtime configuration | `what/configuration.md` |
| Understand multicluster support | `what/multicluster.md` |
| Find code locations | `how/project-structure.md` |

## Cross-Reference

| what/ | how/ |
|---|---|
| `what/system-overview.md` | `how/project-structure.md` |
| `what/poll-loop.md` | `how/project-structure.md` (adapter package) |
| `what/agenticrun-building.md` | `how/project-structure.md` (agenticrun package) |
| `what/alert-retrieval.md` | `how/project-structure.md` (alertmanager package) |
| `what/configuration.md` | `how/project-structure.md` (config package) |
| `what/multicluster.md` | `how/project-structure.md` (multicluster package) |

## Conventions

- **Rule numbering:** behavioral rules are numbered sequentially within each what/ file. Numbers are stable identifiers; do not renumber when a rule is removed (leave a gap) or inserted (use sub-numbers like 16a, 16b). This keeps external references (Jira comments, PR descriptions) valid.
- **Planned changes lifecycle:** unimplemented behavior is marked `[PLANNED]` or `[PLANNED: TICKET-XXXX]` inline next to the rule it affects. When implemented: update the rule text and change the marker to `[DONE: TICKET-XXXX]`. `[DONE]` markers are cleanup candidates.
- **Constraints:** component-specific constraints go in the relevant what/ file's Constraints section. Development conventions go in CLAUDE.md.
- **Authority:** what/ specs are authoritative for behavior. how/ specs are authoritative for implementation. When they conflict, what/ wins.
- **When to create a new file vs. extend an existing one:** if the concern has its own lifecycle, configuration surface, and can be understood independently, it gets its own file. If it's a capability added to an existing component, it goes in that component's file.

## Project Context

This repo is part of the OpenShift Lightspeed family. See `ols/.ai/spec/README.md` for the product-level spec index and `ols/.ai/spec/how/repo-map.md` for the cross-repo concern lookup table.
