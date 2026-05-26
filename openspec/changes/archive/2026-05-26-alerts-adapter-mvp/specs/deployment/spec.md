## ADDED Requirements

### Requirement: Run as a single-replica Deployment
The adapter SHALL run as a single-replica Deployment in the `openshift-lightspeed` namespace with ServiceAccount `lightspeed-agentic-alerts-adapter`.

#### Scenario: Deployment configuration
- **WHEN** the adapter is deployed
- **THEN** it runs as a Deployment with 1 replica in namespace `openshift-lightspeed`
- **AND** uses ServiceAccount `lightspeed-agentic-alerts-adapter`

### Requirement: Provide health and readiness probes
The adapter SHALL expose an HTTP server on port 8081 with `/healthz` (liveness) and `/readyz` (readiness) endpoints. The liveness probe SHALL return 200 if the process is running. The readiness probe SHALL return 200 after the first successful AlertManager poll.

#### Scenario: Liveness probe
- **WHEN** a GET request is sent to `/healthz` on port 8081
- **THEN** the adapter returns HTTP 200 if the process is running

#### Scenario: Readiness before first poll
- **WHEN** a GET request is sent to `/readyz` before the first successful AlertManager poll
- **THEN** the adapter returns a non-200 status

#### Scenario: Readiness after first poll
- **WHEN** a GET request is sent to `/readyz` after the first successful AlertManager poll
- **THEN** the adapter returns HTTP 200

### Requirement: RBAC for AlertManager access
The adapter SHALL have a RoleBinding in `openshift-monitoring` granting `monitoring-alertmanager-view` to its ServiceAccount, enabling read access to AlertManager.

#### Scenario: AlertManager RBAC
- **WHEN** the adapter is deployed
- **THEN** a RoleBinding exists in `openshift-monitoring` binding role `monitoring-alertmanager-view` to ServiceAccount `lightspeed-agentic-alerts-adapter` in namespace `openshift-lightspeed`

### Requirement: RBAC for Proposal management
The adapter SHALL have a ClusterRole and ClusterRoleBinding granting `create`, `list`, and `get` verbs on `proposals` in the `agentic.openshift.io` API group.

#### Scenario: Proposal RBAC
- **WHEN** the adapter is deployed
- **THEN** a ClusterRole exists with verbs `create`, `list`, `get` on resource `proposals` in API group `agentic.openshift.io`
- **AND** a ClusterRoleBinding binds it to ServiceAccount `lightspeed-agentic-alerts-adapter` in namespace `openshift-lightspeed`

### Requirement: Container image
The adapter SHALL include a Containerfile to build the adapter image.

#### Scenario: Container build
- **WHEN** the Containerfile is built
- **THEN** it produces a container image with the adapter binary as the entrypoint

### Requirement: Structured JSON logging
The adapter SHALL use `log/slog` with JSON output. Info level SHALL log poll cycles, Proposal creation, and startup/shutdown. Error level SHALL log API failures. Debug level SHALL log skipped alerts with reasons.

#### Scenario: Proposal creation logged
- **WHEN** a Proposal is created
- **THEN** the adapter logs at info level with the alert name, namespace, and Proposal name

#### Scenario: Skipped alert logged
- **WHEN** an alert is skipped due to deduplication
- **THEN** the adapter logs at debug level with the fingerprint and skip reason
