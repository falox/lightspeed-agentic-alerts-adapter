# AgenticRun Building

Translates AlertManager alerts into AgenticRun custom resources with deterministic naming, stable fingerprinting for deduplication, Kubernetes-safe metadata, and a templated request for the analysis agent.

## Behavioral Rules

### CR Construction

1. The adapter SHALL convert an AlertManager `GettableAlert` into an `AgenticRun` CR with deterministic naming, Kubernetes-safe metadata, and a templated request.
2. Two fingerprint labels SHALL be set on each AgenticRun: `agentic.openshift.io/alert-fingerprint` with the original AlertManager fingerprint (truncated to 8 characters) for UI lookups, and `agentic.openshift.io/alert-group-id` with the stable fingerprint for deduplication.
3. When the alert has a `namespace` label, the AgenticRun name SHALL be `{alertname}-{namespace}-{startsAt_hash}` and `spec.targetNamespaces` SHALL be set to `[namespace]`.
4. When the alert has no `namespace` label, the AgenticRun name SHALL be `{alertname}-{startsAt_hash}` and `spec.targetNamespaces` SHALL be omitted.
5. The same alert passed to Build twice SHALL produce AgenticRuns with identical names, enabling Kubernetes 409 deduplication for the exact same alert instance.
6. When equivalent alerts from two reconciliation targets are built with different target identities, their AgenticRuns SHALL have distinct deterministic names, while repeated builds for either target identity SHALL retain the same name.

### Stable Fingerprint (Scope Hashing)

7. The adapter SHALL compute a stable fingerprint by removing a configurable set of ignored labels from the alert's label set, sorting the remaining `key=value` pairs lexicographically, joining them with a null byte (`\0`) separator, and hashing with FNV-64a truncated to 8 hex characters. The null byte is safe because Prometheus label names and values cannot contain null bytes.
8. Two alerts differing only in ignored labels (e.g., different pod names) SHALL produce the same stable fingerprint.
9. Two alerts differing in non-ignored labels SHALL produce different stable fingerprints (hash collisions are statistically negligible).
10. When the ignored labels list is empty, all alert labels SHALL be included in the hash.
11. When an alert has a nil AlertManager fingerprint, the adapter SHALL return an error.

### EmergencyStopped Replacement

12. The adapter SHALL support creating a replacement AgenticRun for a still-firing alert when the original alert-derived name already exists for an EmergencyStopped run and the alert is eligible for creation.
13. A replacement AgenticRun SHALL use the next deterministic retry name, which SHALL remain within the Kubernetes name length limit.
14. The same set of existing AgenticRuns evaluated for the same alert instance SHALL always produce the same next retry name.

### Metadata Sanitization

15. Characters not allowed in DNS subdomain names in the alertname or namespace SHALL be replaced with hyphens and lowercased.
16. When the computed AgenticRun name would exceed 63 characters, the alertname component SHALL be truncated to fit while preserving the namespace and startsAt hash suffix.
17. Label values exceeding 63 characters SHALL be truncated to 63 characters and trimmed of trailing non-alphanumeric characters.
18. Invalid characters in label values SHALL be replaced with hyphens; leading/trailing non-alphanumeric characters SHALL be trimmed.

### Request Template

19. The adapter SHALL render `spec.request` using an embedded Go template that includes the alert name, severity, namespace, description, and runbook URL. Only allow-listed fields SHALL be passed to the template; the full Labels map SHALL NOT be included.
20. When the alert has summary and description annotations, both SHALL be included in the rendered request.
21. When the alert has no summary or description annotations, the corresponding fields SHALL be empty and no error returned.
22. Unicode control characters (except newline), Unicode format characters, and backtick runs of 3 or more SHALL be stripped from alert data before template rendering.
23. Single and double backticks SHALL be preserved.
24. Extra labels beyond allow-listed fields (alertname, severity, namespace) SHALL NOT appear in the rendered request.
25. When shared skills are configured with paths, the rendered request SHALL contain a skill hint listing those paths (prefixed with `/app`).
26. When no run-level skills are configured, the rendered request SHALL contain the generic investigation instruction instead of a skill hint.

### Workflow Steps

27. The adapter SHALL set analysis, execution, and verification steps on the AgenticRun, each referencing the `default` agent.
28. When shared skills are configured, `spec.tools.skills` SHALL contain the configured entries with their images and paths.
29. When no run-level skills are configured, `spec.tools` SHALL be omitted (zero value).

### AgenticRun CRUD

30. `ListAgenticRuns` SHALL list AgenticRun CRs filtered by the `agentic.openshift.io/source=alertmanager` label to support deduplication queries.
31. When listing for the local target and an Alertmanager-created AgenticRun has no target-identity label, the adapter SHALL return that AgenticRun for local deduplication and exclude AgenticRuns bearing another target identity.
32. When the Kubernetes API returns an error during listing, `ListAgenticRuns` SHALL return a wrapped error with context.
33. `CreateAgenticRun` SHALL create the AgenticRun on the hub cluster. For spoke targets, the AgenticRun SHALL carry the spoke target identity label.
34. When the Kubernetes API returns 409 AlreadyExists, `CreateAgenticRun` SHALL log at Info level and return `false, nil`.
35. When the Kubernetes API returns a non-409 error, `CreateAgenticRun` SHALL return `false` and a wrapped error.
