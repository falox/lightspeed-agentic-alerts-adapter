# Poll Loop

The core reconcile loop that continuously polls AlertManager, applies receiver filtering and stateless deduplication, and creates AgenticRun CRs for qualifying alerts. Includes creation backoff to limit repeated failures.

## Behavioral Rules

### Polling

1. The adapter SHALL read operational parameters (`pollInterval`, `preRunDelay`, `postRunDelay`) from the `ConfigSource` at the start of each reconcile cycle and use them for that cycle's filtering and deduplication. The default poll interval is 30 seconds.
2. When the loaded `pollInterval` differs from the current ticker interval, the adapter SHALL reset the ticker to the new interval and log the change.
3. For each configured reconciliation target, the filter order SHALL be: receiver allowlist, pre-run delay, active AgenticRun, post-run delay, creation backoff.

### Suspended Mode

4. At the start of each reconcile cycle, the adapter SHALL read the cluster-scoped `AgenticOLSConfig` singleton named `cluster`; when `spec.suspended` is true, the adapter SHALL skip that cycle before polling AlertManager, listing AgenticRuns, or creating AgenticRuns.
5. When the `AgenticOLSConfig` CRD or singleton object is absent, the adapter SHALL behave as if suspended mode is disabled.
6. When reading the `AgenticOLSConfig` suspension state fails for a reason other than absent CRD or absent singleton, the adapter SHALL log the error and skip the reconcile cycle; the next poll retries.

### Target Reconciliation

7. When suspended mode is disabled and the poll interval elapses, the adapter SHALL fetch alerts from every configured reconciliation target, list hub AgenticRuns matching that target's identity, apply receiver filtering then dedup rules independently for that target, and create target-identified AgenticRuns on the hub for qualifying alerts.
8. When AlertManager returns an error for one target, the adapter SHALL log the target-specific error, skip that target's reconciliation, and continue for remaining targets.
9. When the hub Kubernetes API returns an error during AgenticRun listing or creation for one target, the adapter SHALL log the error, skip the failed operation for that target, and continue for remaining targets.

### Concurrent Target Reconciliation

10. When `--multicluster` is set, the adapter SHALL reconcile independent targets concurrently while limiting simultaneous reconciliations to `MULTICLUSTER_MAX_CONCURRENT_TARGETS` (default 4, must be a positive integer).
11. Alert processing within a target SHALL remain sequential, and the adapter SHALL wait for all started target reconciliations before beginning another poll cycle.
12. When `--multicluster` is not set, the adapter SHALL reconcile the local target sequentially and ignore `MULTICLUSTER_MAX_CONCURRENT_TARGETS`.
13. When `--multicluster` is set and `MULTICLUSTER_MAX_CONCURRENT_TARGETS` is absent or not a valid positive integer, the adapter SHALL fail startup with an error.

### Target Timeout

14. The adapter SHALL apply the configured `pollInterval` as a deadline to every target reconciliation. AlertManager and hub Kubernetes operations for a target SHALL use that deadline while preserving cancellation from the parent context.
15. When an operation for a target does not complete within `pollInterval`, the target reconciliation context SHALL be cancelled and remaining targets SHALL continue.

### Receiver Filtering

16. The adapter SHALL skip any alert whose AlertManager receivers do not include at least one entry from the configured `allowedReceivers` list. Comparison SHALL be case-insensitive.
17. When an alert has no receivers or an empty receivers list, the alert SHALL be skipped and logged at Debug level.
18. When the allowlist is empty (explicitly `[]`), all alerts are skipped.
19. The adapter SHALL log the effective `allowedReceivers` list at Info level at startup and when configuration is reloaded.

### Pre-Run Delay (Skip Transient Alerts)

20. The adapter SHALL not create an AgenticRun for an alert that has been firing for less than the configured `preRunDelay`, to filter out transient alerts that resolve on their own.
21. When `preRunDelay` is 0, all alerts pass the pre-run delay check regardless of firing duration.

### Active AgenticRun Check

22. The adapter SHALL not create an AgenticRun for an alert that already has an active (non-terminal) AgenticRun, identified by matching the `alert-group-id` label.
23. `EmergencyStopped` SHALL be treated as terminal, not active.

### Post-Run Delay (Cooldown)

24. The adapter SHALL not create an AgenticRun for an alert that has a terminal AgenticRun (Completed, Failed, Denied, Escalated, EmergencyStopped) within the configured `postRunDelay` (default 1h), to avoid repeated analysis of recently resolved alerts.
25. The terminal time is determined from the `LastTransitionTime` of the condition that caused the run to reach its terminal phase.
26. When `postRunDelay` is 0, all alerts pass the post-run delay check.

### Creation Backoff

27. The adapter SHALL defer repeated AgenticRun creation attempts after a creation error. Backoff state SHALL be isolated by reconciliation target and stable alert-group identity, and SHALL increase exponentially from an initial delay (1 minute) to a bounded maximum (10 minutes).
28. When a creation attempt succeeds after a target and alert group entered backoff, the adapter SHALL clear its backoff state so a future failure begins at the initial delay.
29. An AlreadyExists response SHALL retain its existing no-op behavior and SHALL NOT advance backoff.
30. Errors outside AgenticRun creation (e.g., alert retrieval failures) SHALL NOT use creation backoff.
31. The adapter SHALL log when an alert group enters creation backoff, when its delay increases, and when backoff is cleared, including the target, alert identity, and applicable duration.

### Graceful Shutdown

32. The adapter SHALL exit cleanly on SIGTERM or SIGINT, completing or cancelling any in-flight poll cycle before stopping, and exit with status code 0.
