# System Overview

The alerts adapter is a stateless, single-replica Go component that polls OpenShift AlertManager for firing alerts and creates `AgenticRun` custom resources (`agentic.openshift.io/v1alpha1`) to trigger automated remediation via the Lightspeed Agentic operator. It uses a create-only, poll-based design with no internal state: each cycle diffs AlertManager against the Kubernetes API to determine which alerts need action.

## Behavioral Rules

### System Role

1. The adapter SHALL poll AlertManager on a configurable interval (default 30s), fetch active alerts, and create `AgenticRun` CRs for alerts that pass filtering and deduplication.
2. The adapter SHALL operate as a single-replica deployment inside an OpenShift cluster in the `openshift-lightspeed` namespace.
3. The adapter SHALL use polling (not webhooks) so a restart immediately sees all firing alerts without requiring redelivery.
4. The adapter SHALL create all AgenticRuns in the `openshift-lightspeed` namespace (or the namespace from `POD_NAMESPACE`).
5. The adapter SHALL treat HTTP 409 AlreadyExists on AgenticRun creation as a no-op (return `false, nil`).

### Component Inventory

6. **AlertManager client** (`internal/alertmanager`): retrieves active, non-silenced, non-inhibited alerts. Re-reads the ServiceAccount bearer token on every call to handle rotation. Trusts the in-cluster service CA for TLS.
7. **AgenticRun builder and client** (`internal/agenticrun`): `build.go` translates an alert into an AgenticRun CR with deterministic naming and embedded Go template for the request field. `client.go` wraps controller-runtime to create/list AgenticRuns.
8. **Poll loop** (`internal/adapter`): runs the `reconcile` method on each tick. Stateless deduplication: skips alerts below `preRunDelay`, with an active (non-terminal) AgenticRun, within `postRunDelay` of a terminal AgenticRun, or within creation backoff. Filters by receiver allowlist.
9. **Configuration** (`internal/config`): loads runtime-tunable parameters from a ConfigMap each cycle.
10. **Suspended mode client** (`internal/agenticolsconfig`): reads the `AgenticOLSConfig` singleton to determine whether the adapter should skip reconciliation.
11. **Multicluster** (`internal/multicluster`): watches `SpokeCluster` resources, builds per-cluster reconciliation targets, manages a dynamic target registry.

### External Dependencies

12. The adapter depends on the `AgenticRun` CRD types from `github.com/openshift/lightspeed-agentic-operator/api`.
13. The adapter depends on the `SpokeCluster` CRD types from `github.com/openshift/lightspeed-hub/api` (multicluster mode only).
14. The adapter uses the AlertManager v2 API client from `github.com/prometheus/alertmanager`.

### Lifecycle

15. The adapter SHALL exit cleanly on SIGTERM or SIGINT, completing or cancelling any in-flight poll cycle before stopping.
16. The adapter SHALL use structured JSON logging via `log/slog`, with the logger passed explicitly (no globals except `slog.SetDefault` in main).

## Configuration Surface

| Field | Default | Description |
|---|---|---|
| `pollInterval` | `30s` | Interval between reconcile cycles |
| `preRunDelay` | `0s` | Minimum alert firing duration before creating an AgenticRun |
| `postRunDelay` | `1h` | Cooldown after a terminal AgenticRun before creating another |
| `filtering.allowedReceivers` | `[]` (empty) | AlertManager receiver allowlist; empty means no alerts processed |
| `deduplication.ignoredLabels` | `[pod, instance, endpoint, uid]` | Labels excluded from stable fingerprint computation |
| `tools.skills` | (none) | Run-level skills (OCI images + mount paths) added to AgenticRuns |
| `ALERTMANAGER_URL` | `https://alertmanager-main.openshift-monitoring.svc:9094` | AlertManager endpoint (env var) |
| `POD_NAMESPACE` | `openshift-lightspeed` | Namespace for ConfigMap and AgenticRun creation (env var) |
| `--multicluster` | `false` | Enable SpokeCluster discovery and per-cluster reconciliation (flag) |
| `MULTICLUSTER_MAX_CONCURRENT_TARGETS` | `4` | Max simultaneous target reconciliations in multicluster mode (env var) |

## Constraints

- Terminal phases: Completed, Failed, Denied, Escalated, EmergencyStopped.
- AgenticRun names are bounded to 63 characters (Kubernetes label value limit, since the agentic operator uses the name as a label value).
- Two fingerprint labels on each AgenticRun: `alert-fingerprint` (original AlertManager fingerprint for UI lookups) and `alert-group-id` (stable FNV-64a hash for dedup matching).
