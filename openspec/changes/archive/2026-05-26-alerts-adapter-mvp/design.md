## Context

This is a greenfield Go 1.26 component — the `lightspeed-agentic-alerts-adapter` repo currently contains only LICENSE and OWNERS. The adapter runs alongside the Lightspeed Agentic Operator in an OpenShift cluster, creating Proposal CRs from firing alerts. The operator already watches Proposals across all namespaces and reconciles them through analysis, execution, and verification steps.

AlertManager runs in `openshift-monitoring` and exposes a stable v2 API. The adapter runs in `openshift-lightspeed` as a single-replica Deployment.

## Goals / Non-Goals

**Goals:**

- Poll AlertManager for firing alerts and create Proposal CRs
- Prevent duplicate Proposals via deterministic naming, initial delay, and cooldown
- Run as a stateless, single-replica Deployment with no persistent storage
- Authenticate using the pod's ServiceAccount token (no external secrets)
- Provide structured JSON logging and health/readiness probes

**Non-Goals:**

- Alert filtering by severity, label selectors, or opt-in labels
- Configurable parameters (poll interval, delays, cooldown) — all are Go constants for now
- Prometheus metrics
- Multi-replica / leader election
- Adapter-specific `analysisOutput.schema`
- Per-alert workflow selection (advisory vs full remediation)
- Alert grouping — each alert maps 1:1 to a Proposal

## Decisions

### Project structure

```
lightspeed-agentic-alerts-adapter/
├── cmd/
│   └── adapter/
│       └── main.go
├── internal/
│   ├── alertmanager/       # AlertManager v2 API client
│   │   ├── client.go       # HTTP client, auth, TLS
│   │   └── types.go        # Alert payload structs
│   ├── adapter/            # Core poll-diff-create loop
│   │   ├── adapter.go      # Main loop orchestration
│   │   ├── dedup.go        # Deduplication logic
│   │   └── mapper.go       # Alert → Proposal mapping
│   └── proposal/           # Proposal CR helpers
│       └── builder.go      # Build typed Proposal objects
├── deploy/                 # OpenShift manifests
│   ├── deployment.yaml
│   ├── rbac.yaml
│   └── serviceaccount.yaml
├── Containerfile
├── Makefile
├── go.mod
└── go.sum
```

The `internal/alertmanager` package owns the HTTP interaction with AlertManager. The `internal/adapter` package owns the core loop: poll, diff, create. The `internal/proposal` package builds typed Proposal objects using the operator's API types.

`main.go` wires everything together: in-cluster config, Kubernetes client, AlertManager client, signal handling, and the poll loop.

### AlertManager client

Direct HTTP client using `net/http` — no third-party AlertManager SDK. The v2 API is a single GET endpoint with query parameters. The client:

- Calls `GET https://alertmanager-main.openshift-monitoring.svc:9094/api/v2/alerts?active=true&silenced=false&inhibited=false`
- Authenticates with Bearer token from `/var/run/secrets/kubernetes.io/serviceaccount/token`
- Verifies TLS against the cluster CA bundle at `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`
- Parses the JSON response into typed Go structs (only the fields we need: `status`, `labels`, `annotations`, `startsAt`, `fingerprint`)

**Why not a generated client or SDK**: The AlertManager v2 API is simple (one endpoint, one method). A generated client would add dependency weight for no benefit.

### Kubernetes client

Uses `controller-runtime`'s client (`sigs.k8s.io/controller-runtime/pkg/client`) rather than raw `client-go`. This gives us:

- Typed access to Proposal CRs using the operator's `api/v1alpha1` types
- Built-in scheme registration
- Consistent with what the operator itself uses

The adapter imports `github.com/openshift/lightspeed-agentic-operator/api/v1alpha1` for the Proposal Go types. It uses `rest.InClusterConfig()` for authentication.

### Poll loop

The main loop runs on a 30-second ticker. Each cycle:

1. **Fetch alerts** — GET from AlertManager, parse response
2. **List existing Proposals** — LIST with label selector `agentic.openshift.io/source=alertmanager`
3. **Diff** — for each alert, check three skip conditions:
   - `now - alert.startsAt < InitialDelay (5 min)` → skip, too transient
   - Active Proposal with matching fingerprint exists → skip, already handling
   - Terminal Proposal with matching fingerprint within `CooldownWindow (1 hour)` → skip, too soon
4. **Create** — build and create a Proposal CR for alerts that pass all checks

"Active" means a Proposal whose derived phase is not terminal. Terminal phases are: Completed, Failed, Denied, Escalated. The adapter uses `DerivePhase()` from the operator's API package to compute phase from conditions.

### Deterministic Proposal naming

Format: `{alertname}-{namespace}-{fingerprint[:8]}`

- `alertname`: from `alert.labels["alertname"]`, lowercased
- `namespace`: from `alert.labels["namespace"]`, omitted if empty (cluster-scoped)
- `fingerprint`: first 8 characters of AlertManager's fingerprint

Sanitization: replace non-alphanumeric characters (except `-`) with `-`, collapse consecutive hyphens, trim leading/trailing hyphens, truncate to 253 characters (DNS subdomain limit per RFC 1123).

Examples:
- `kubepodcrashlooping-production-a1b2c3d4`
- `etcdhighfsyncdurations-f9e8d7c6` (cluster-scoped, no namespace segment)

This naming makes creation idempotent: Kubernetes returns 409 Conflict if the name already exists, which the adapter treats as success.

### Alert-to-Proposal mapping

**Namespace**: the alert's `namespace` label. If absent (cluster-scoped alert), falls back to `openshift-lightspeed`.

**`spec.request`**: rendered from a hardcoded Go `text/template`:

```
Alert: {{.AlertName}} (severity: {{.Severity}})
Namespace: {{.Namespace}}

{{.Description}}

Investigate the root cause and propose a remediation.

Alert labels:
{{range $k, $v := .Labels}}- {{$k}}: {{$v}}
{{end}}
```

**`spec.targetNamespaces`**: `[alert.labels["namespace"]]` if present, empty if cluster-scoped.

**`spec.analysis`**: `{agent: "default"}`
**`spec.execution`**: `{agent: "default"}`
**`spec.verification`**: `{agent: "default"}`
**`spec.analysisOutput`**: omitted (defaults to `mode: Default`, no custom schema)

**Labels**:
- `agentic.openshift.io/source: alertmanager`
- `agentic.openshift.io/alert-fingerprint: <fingerprint[:8]>`
- `agentic.openshift.io/alert-name: <alertname>`
- `agentic.openshift.io/alert-severity: <severity>`

**Annotations**:
- `agentic.openshift.io/alert-starts-at: <RFC3339>`
- `agentic.openshift.io/alert-summary: <summary, truncated>`

### Error handling

All errors are non-fatal. The adapter logs and retries on the next poll cycle:

| Failure | Behavior |
|---------|----------|
| AlertManager unreachable | Log error, skip entire cycle |
| Kubernetes API unreachable | Log error, skip Proposal creation |
| Proposal creation fails (non-409) | Log error with alert details, skip alert (retried next cycle since no Proposal exists) |
| 409 Conflict | Treated as success (Proposal already exists) |
| Invalid alert data (missing alertname, template error) | Log and skip individual alert, continue processing others |

### Health probes

A minimal HTTP server on `:8081`:
- `/healthz` — liveness: returns 200 if the process is running
- `/readyz` — readiness: returns 200 after the first successful AlertManager poll

### Logging

`log/slog` with JSON output:

| Level | What |
|-------|------|
| Info | Poll cycle start/end (with counts), Proposal created (alertname, namespace, proposal name), startup/shutdown |
| Error | AlertManager unreachable, K8s API errors, creation failures |
| Debug | Alerts skipped (with reason: initial-delay, active-proposal, cooldown) |

### Configuration constants

All configuration is hardcoded as Go constants:

| Constant | Value |
|----------|-------|
| `PollInterval` | `30s` |
| `InitialDelay` | `5m` |
| `CooldownWindow` | `1h` |
| `AlertManagerURL` | `https://alertmanager-main.openshift-monitoring.svc:9094` |
| `DefaultNamespace` | `openshift-lightspeed` |
| `DefaultAgent` | `"default"` |

### Deployment

Single-replica Deployment in `openshift-lightspeed`:
- ServiceAccount: `lightspeed-agentic-alerts-adapter`
- Image: `quay.io/openshift-lightspeed/lightspeed-agentic-alerts-adapter:latest`
- Probes: liveness and readiness on `:8081`

RBAC:
- RoleBinding in `openshift-monitoring` → `monitoring-alertmanager-view` (read alerts)
- ClusterRole + ClusterRoleBinding → `create`, `list`, `get` on `proposals.agentic.openshift.io`

## Risks / Trade-offs

**Single replica means brief blind spots during restarts** → Acceptable: the stateless poll-based design catches up immediately on the next cycle. No alerts are permanently missed — they just wait up to 30 seconds (one poll interval).

**ClusterRoleBinding for Proposal creation is broad** → Required for namespace-locality (Proposals live in the alert's namespace). The adapter only has `create`, `list`, `get` — no `update`, `delete`, or `patch`.

**Importing the operator's API types creates a dependency coupling** → The Proposal CRD is the contract. Using typed Go structs catches breaking changes at compile time rather than runtime. The alternative (unstructured client) would be more fragile.

**Deterministic naming with 8-char fingerprint prefix has a collision risk** → Extremely low in practice. AlertManager fingerprints are hex-encoded hashes; 8 hex chars = 4 billion possible values per alertname+namespace combination.

**No metrics in MVP** → Logging provides basic observability. Metrics are deferred to keep scope tight but are planned for a future iteration.
