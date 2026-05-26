## Why

The Lightspeed Agentic Operator reconciles Proposal CRs to drive AI-assisted remediation, but nothing creates those Proposals automatically from cluster alerts. Cluster admins must manually observe alerts and create Proposals by hand. This adapter closes the loop by automatically translating firing alerts into Proposals, enabling hands-off alert remediation.

The adapter polls AlertManager rather than receiving webhooks: if the adapter is down when a webhook arrives, the notification is lost, whereas polling immediately sees all firing alerts on the next cycle — zero missed alerts, zero catch-up delay, and no AlertManager configuration changes required. It integrates at AlertManager (not Thanos Ruler) because AlertManager provides a stable API with full alert metadata and handles grouping, silencing, and inhibition — integrating upstream would mean reimplementing that logic.

## What Changes

- New Go 1.26 binary that runs as a single-replica Deployment in `openshift-lightspeed`
- Polls AlertManager (`GET /api/v2/alerts?active=true&silenced=false&inhibited=false`) every 30 seconds
- Creates `agentic.openshift.io/v1alpha1` Proposal CRs for firing alerts that pass deduplication
- Deduplication via deterministic naming (`{alertname}-{namespace}-{fingerprint[:8]}`), initial delay (5 min), and cooldown window (1 hour after terminal Proposals). The fingerprint is the one provided by AlertManager (a hash of the alert's label set), not computed by the adapter.
- Proposals created in the alert's source namespace (cluster-scoped alerts fall back to `openshift-lightspeed`)
- Full remediation workflow: analysis + execution + verification, all using the `default` agent
- Request text composed from a hardcoded Go `text/template` with alert metadata
- 1:1 alert-to-Proposal cardinality — each unique alert (by fingerprint) maps to exactly one Proposal, no grouping
- When an alert resolves while its Proposal is active, the adapter does nothing — the Proposal runs to completion (self-resolution doesn't confirm the fix, and the analysis is still valuable)
- Stateless, idempotent, create-only — never modifies or deletes Proposals
- Structured JSON logging via `log/slog`, health/readiness probes on `:8081`
- Authenticates to AlertManager using the pod's ServiceAccount token

## Capabilities

### New Capabilities

- `alert-polling`: Polling AlertManager for active, non-silenced, non-inhibited alerts and parsing the response
- `deduplication`: Preventing duplicate Proposals via deterministic naming, initial delay, and cooldown window
- `proposal-creation`: Mapping alert data to Proposal CRs and creating them via the Kubernetes API
- `deployment`: Container image, Deployment manifest, RBAC, ServiceAccount, health probes

### Modified Capabilities

None — this is a new component.

## Impact

- **Dependencies**: `lightspeed-agentic-operator/api` (Proposal types), `k8s.io/client-go`, AlertManager v2 API
- **RBAC**: Requires read access to AlertManager in `openshift-monitoring` and create/list/get on Proposals cluster-wide
- **Cluster**: Adds a Deployment in `openshift-lightspeed` and a ClusterRoleBinding
- **Operator**: No changes to the operator — the adapter is a separate component that creates CRs the operator already knows how to reconcile
