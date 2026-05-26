## ADDED Requirements

### Requirement: Skip alerts within initial delay
The adapter SHALL skip any alert whose `startsAt` timestamp is less than 5 minutes ago. This filters transient alerts that may resolve on their own.

#### Scenario: Alert firing for less than 5 minutes
- **WHEN** an alert has `startsAt` of 3 minutes ago
- **THEN** the adapter skips the alert
- **AND** logs the skip at debug level with reason "initial-delay"

#### Scenario: Alert firing for more than 5 minutes
- **WHEN** an alert has `startsAt` of 6 minutes ago
- **AND** no other skip conditions apply
- **THEN** the adapter proceeds to create a Proposal

### Requirement: Skip alerts with an active Proposal
The adapter SHALL skip any alert for which a non-terminal Proposal already exists, identified by matching the `agentic.openshift.io/alert-fingerprint` label. Terminal phases are: Completed, Failed, Denied, Escalated.

#### Scenario: Active Proposal exists for alert
- **WHEN** an alert has fingerprint `abc123` and a Proposal with label `agentic.openshift.io/alert-fingerprint: abc123` exists in phase Analyzing
- **THEN** the adapter skips the alert
- **AND** logs the skip at debug level with reason "active-proposal"

#### Scenario: No Proposal exists for alert
- **WHEN** an alert has fingerprint `abc123` and no Proposal with label `agentic.openshift.io/alert-fingerprint: abc123` exists
- **AND** no other skip conditions apply
- **THEN** the adapter proceeds to create a Proposal

### Requirement: Skip alerts within cooldown window
The adapter SHALL skip any alert for which a terminal Proposal exists and that Proposal reached its terminal state less than 1 hour ago. The terminal timestamp is derived from the Proposal's condition timestamps.

#### Scenario: Terminal Proposal within cooldown
- **WHEN** an alert has fingerprint `abc123` and a Proposal with the same fingerprint reached phase Completed 30 minutes ago
- **THEN** the adapter skips the alert
- **AND** logs the skip at debug level with reason "cooldown"

#### Scenario: Terminal Proposal outside cooldown
- **WHEN** an alert has fingerprint `abc123` and a Proposal with the same fingerprint reached phase Failed 2 hours ago
- **AND** no other skip conditions apply
- **THEN** the adapter proceeds to create a Proposal

### Requirement: Deterministic Proposal naming
The adapter SHALL name Proposals using the pattern `{alertname}-{namespace}-{fingerprint[:8]}`, where alertname and namespace come from alert labels and fingerprint is the first 8 characters of AlertManager's fingerprint. For cluster-scoped alerts (no namespace label), the namespace segment SHALL be omitted. Names SHALL be sanitized to DNS subdomain rules (RFC 1123): lowercased, non-alphanumeric characters replaced with hyphens, consecutive hyphens collapsed, leading/trailing hyphens trimmed, truncated to 253 characters.

#### Scenario: Namespaced alert naming
- **WHEN** an alert has alertname `KubePodCrashLooping`, namespace `production`, and fingerprint `a1b2c3d4e5f6`
- **THEN** the Proposal is named `kubepodcrashlooping-production-a1b2c3d4`

#### Scenario: Cluster-scoped alert naming
- **WHEN** an alert has alertname `EtcdHighFsyncDurations`, no namespace label, and fingerprint `f9e8d7c6b5a4`
- **THEN** the Proposal is named `etcdhighfsyncdurations-f9e8d7c6`

#### Scenario: Duplicate creation attempt
- **WHEN** the adapter attempts to create a Proposal and Kubernetes returns 409 Conflict
- **THEN** the adapter treats it as success (the Proposal already exists)

### Requirement: Stateless deduplication
The adapter SHALL maintain no in-memory or persistent state for deduplication. Each poll cycle SHALL recompute the full diff between AlertManager's firing alerts and existing Proposals in the Kubernetes API.

#### Scenario: Adapter restarts mid-cycle
- **WHEN** the adapter pod restarts
- **THEN** the next poll cycle correctly identifies which alerts need Proposals by querying both AlertManager and the Kubernetes API
- **AND** no alerts are missed and no duplicates are created
