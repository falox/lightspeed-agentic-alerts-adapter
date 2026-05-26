## ADDED Requirements

### Requirement: Create Proposals in the alert's namespace
The adapter SHALL create each Proposal CR in the Kubernetes namespace matching the alert's `namespace` label. For cluster-scoped alerts (no `namespace` label), the adapter SHALL create the Proposal in `openshift-lightspeed`.

#### Scenario: Namespaced alert
- **WHEN** an alert has label `namespace: my-app`
- **THEN** the Proposal is created in namespace `my-app`

#### Scenario: Cluster-scoped alert
- **WHEN** an alert has no `namespace` label
- **THEN** the Proposal is created in namespace `openshift-lightspeed`

### Requirement: Set target namespaces from alert
The adapter SHALL set `spec.targetNamespaces` to a single-element list containing the alert's `namespace` label. For cluster-scoped alerts, `spec.targetNamespaces` SHALL be left empty.

#### Scenario: Namespaced alert target
- **WHEN** an alert has label `namespace: my-app`
- **THEN** `spec.targetNamespaces` is set to `["my-app"]`

#### Scenario: Cluster-scoped alert target
- **WHEN** an alert has no `namespace` label
- **THEN** `spec.targetNamespaces` is omitted

### Requirement: Compose request text from alert data
The adapter SHALL compose `spec.request` using a hardcoded Go `text/template` that includes the alert name, severity, namespace, description, and all labels. The template SHALL include the instruction "Investigate the root cause and propose a remediation."

#### Scenario: Request text for a namespaced alert
- **WHEN** an alert has alertname `KubePodCrashLooping`, severity `warning`, namespace `my-app`, and description `Pod is restarting frequently`
- **THEN** `spec.request` contains the alert name, severity, namespace, description, all labels, and the investigation instruction

### Requirement: Configure full remediation workflow
The adapter SHALL configure all three workflow steps — analysis, execution, and verification — each using the `default` agent. The `analysisOutput` field SHALL be omitted (defaulting to mode `Default`).

#### Scenario: Proposal workflow configuration
- **WHEN** a Proposal is created
- **THEN** `spec.analysis` is set with agent `default`
- **AND** `spec.execution` is set with agent `default`
- **AND** `spec.verification` is set with agent `default`

### Requirement: Label Proposals with alert metadata
The adapter SHALL set the following labels on each Proposal:
- `agentic.openshift.io/source: alertmanager`
- `agentic.openshift.io/alert-fingerprint: <fingerprint[:8]>`
- `agentic.openshift.io/alert-name: <alertname, lowercased>`
- `agentic.openshift.io/alert-severity: <severity>`

#### Scenario: Labels on created Proposal
- **WHEN** a Proposal is created for an alert with alertname `KubePodCrashLooping`, severity `warning`, fingerprint `a1b2c3d4e5f6`
- **THEN** the Proposal has label `agentic.openshift.io/source` set to `alertmanager`
- **AND** label `agentic.openshift.io/alert-fingerprint` set to `a1b2c3d4`
- **AND** label `agentic.openshift.io/alert-name` set to `kubepodcrashlooping`
- **AND** label `agentic.openshift.io/alert-severity` set to `warning`

### Requirement: Annotate Proposals with alert context
The adapter SHALL set the following annotations on each Proposal:
- `agentic.openshift.io/alert-starts-at: <RFC3339 timestamp>`
- `agentic.openshift.io/alert-summary: <summary annotation, truncated>`

#### Scenario: Annotations on created Proposal
- **WHEN** a Proposal is created for an alert with startsAt `2026-05-26T10:15:00Z` and summary annotation `Pod is crash looping`
- **THEN** the Proposal has annotation `agentic.openshift.io/alert-starts-at` set to `2026-05-26T10:15:00Z`
- **AND** annotation `agentic.openshift.io/alert-summary` set to `Pod is crash looping`

### Requirement: Ignore alert resolution
The adapter SHALL take no action when an alert resolves while its Proposal is active. The Proposal SHALL continue to completion regardless of alert state.

#### Scenario: Alert resolves during active Proposal
- **WHEN** an alert with fingerprint `abc123` resolves
- **AND** an active Proposal with label `agentic.openshift.io/alert-fingerprint: abc123` exists
- **THEN** the adapter does not modify or delete the Proposal

### Requirement: Create-only lifecycle
The adapter SHALL only create Proposal CRs. It SHALL never update, patch, or delete Proposals. The operator owns the Proposal lifecycle after creation.

#### Scenario: Proposal creation failure
- **WHEN** a Proposal creation fails with a non-409 error
- **THEN** the adapter logs the error
- **AND** retries on the next poll cycle (the alert will still be firing and no Proposal exists)
- **AND** does not attempt to modify any existing resources
