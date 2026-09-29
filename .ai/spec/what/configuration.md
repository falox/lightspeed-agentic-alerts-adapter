# Configuration

Runtime-tunable parameters loaded from a Kubernetes ConfigMap, allowing operational tuning without restarting the adapter.

## Behavioral Rules

### ConfigMap Loading

1. The adapter SHALL read the `alerts-adapter-config` ConfigMap from the adapter's namespace on each reconcile cycle and apply the values for that cycle.
2. When the ConfigMap exists with valid YAML and valid duration values in the `config.yaml` key, the adapter SHALL use those values.
3. When the ConfigMap has valid YAML with only some fields specified, the adapter SHALL use specified values and fall back to defaults for missing fields.
4. When the ConfigMap has an unparseable duration string, the adapter SHALL log an error and return a config load error, causing the reconcile loop to fail.
5. When the ConfigMap has invalid YAML, the adapter SHALL log an error and return a config load error.
6. When the ConfigMap exists but lacks the `config.yaml` key, the adapter SHALL fall back to all default values.
7. When the ConfigMap does not exist at startup, the adapter SHALL start normally using defaults and log at Info level.
8. When the ConfigMap is deleted while the adapter is running, the adapter SHALL revert to defaults on the next cycle and log at Info level.

### Default Values

9. Default values when no ConfigMap is present or fields are missing: `pollInterval=30s`, `preRunDelay=0s`, `postRunDelay=1h`, `allowedReceivers=[]`.

### Duration Clamping

10. When `preRunDelay` is explicitly set to `0s`, the adapter SHALL use `preRunDelay=0s` (no delay).
11. When `postRunDelay` is explicitly set to `0s`, the adapter SHALL use `postRunDelay=0s` (overrides the 1h default, no delay).
12. When `preRunDelay` or `postRunDelay` is negative, the adapter SHALL clamp the value to `0s` (no error logged).

### Namespace Resolution

13. The adapter SHALL read the `POD_NAMESPACE` environment variable for the ConfigMap namespace. If unset, it falls back to `openshift-lightspeed`.

### Structured Configuration Sections

14. The adapter SHALL support `filtering.allowedReceivers` nested under a `filtering` section and `deduplication.ignoredLabels` nested under a `deduplication` section.
15. Top-level `allowedReceivers` SHALL continue to be accepted.

### Ignored Labels

16. The adapter SHALL support a `deduplication.ignoredLabels` field specifying which labels to exclude from the stable fingerprint. When absent, defaults to `[pod, instance, endpoint, uid]`. When explicitly set, the specified list fully replaces the default (no merging).
17. When `ignoredLabels` is set to an empty list, no labels are ignored and all alert labels are included in the stable fingerprint hash.

### Skills Configuration

18. The adapter SHALL parse an optional `tools.skills` key from the ConfigMap. Each entry specifies an OCI image and a list of mount paths, mapping to `spec.tools.skills` on AgenticRuns.
19. When the ConfigMap has no `tools` key, the adapter SHALL have empty skills and no error.
20. When a skills entry has an empty `image` field, that entry SHALL be skipped with a warning logged, and remaining valid entries SHALL still be applied.
21. When a skills entry has a non-empty `image` but empty `paths`, that entry SHALL be skipped with a warning logged.
22. When a skills list contains both valid and invalid entries, only valid entries SHALL be included.

## Configuration Surface

| Field | Default | Description |
|---|---|---|
| `pollInterval` | `30s` | Interval between reconcile cycles |
| `preRunDelay` | `0s` | Minimum alert firing duration before creating an AgenticRun |
| `postRunDelay` | `1h` | Cooldown after a terminal AgenticRun |
| `filtering.allowedReceivers` | `[]` | Receiver allowlist |
| `deduplication.ignoredLabels` | `[pod, instance, endpoint, uid]` | Labels excluded from fingerprint |
| `tools.skills[].image` | (none) | OCI image for run-level skill |
| `tools.skills[].paths` | (none) | Mount paths for run-level skill |
| `POD_NAMESPACE` | `openshift-lightspeed` | Namespace for ConfigMap lookup (env var) |
