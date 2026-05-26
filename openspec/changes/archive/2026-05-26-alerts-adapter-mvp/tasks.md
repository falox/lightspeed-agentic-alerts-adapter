## 1. Project Setup

- [x] 1.1 Initialize Go module (`go.mod` with Go 1.26), add dependencies: `lightspeed-agentic-operator/api`, `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`
- [x] 1.2 Create project directory structure: `cmd/adapter/`, `internal/alertmanager/`, `internal/adapter/`, `internal/proposal/`, `deploy/`
- [x] 1.3 Create `Makefile` with targets: `build`, `test`, `lint`, `image`

## 2. AlertManager Client

- [x] 2.1 Define alert types in `internal/alertmanager/types.go` (labels, annotations, startsAt, fingerprint, status)
- [x] 2.2 Implement HTTP client in `internal/alertmanager/client.go`: GET with query params, Bearer token auth, TLS verification, JSON parsing
- [x] 2.3 Write unit tests for alert parsing (valid response, empty response, invalid alert data)

## 3. Deduplication

- [x] 3.1 Implement deterministic naming in `internal/adapter/dedup.go`: `{alertname}-{namespace}-{fingerprint[:8]}` with DNS sanitization
- [x] 3.2 Implement initial delay check (`now - startsAt < 5m`)
- [x] 3.3 Implement active Proposal check (non-terminal Proposal with matching fingerprint label)
- [x] 3.4 Implement cooldown window check (terminal Proposal within 1 hour, using condition timestamps)
- [x] 3.5 Write unit tests for naming (namespaced, cluster-scoped, sanitization, truncation) and all skip conditions

## 4. Proposal Creation

- [x] 4.1 Implement Proposal builder in `internal/proposal/builder.go`: construct typed Proposal from alert data with all labels, annotations, target namespaces, and workflow steps
- [x] 4.2 Implement request template in `internal/adapter/mapper.go`: hardcoded Go `text/template` with alert name, severity, namespace, description, labels, and investigation instruction
- [x] 4.3 Write unit tests for Proposal building (namespaced alert, cluster-scoped alert, label/annotation values) and template rendering

## 5. Core Adapter Loop

- [x] 5.1 Implement poll-diff-create loop in `internal/adapter/adapter.go`: fetch alerts, list Proposals, apply skip conditions, create Proposals
- [x] 5.2 Handle 409 Conflict as success, log non-409 errors and skip individual alert
- [x] 5.3 Handle AlertManager and Kubernetes API unreachable (log error, skip cycle)
- [x] 5.4 Write integration-style tests for the adapter loop (mock both AlertManager and Kubernetes API)

## 6. Entry Point and Health Probes

- [x] 6.1 Implement `cmd/adapter/main.go`: in-cluster config, Kubernetes client, AlertManager client, signal handling, poll loop startup
- [x] 6.2 Implement health server on `:8081` with `/healthz` (always 200) and `/readyz` (200 after first successful poll)
- [x] 6.3 Implement structured JSON logging with `log/slog` (info: poll cycles and creation, error: API failures, debug: skipped alerts)

## 7. Deployment Manifests

- [x] 7.1 Create `Containerfile` (multi-stage build: Go build → distroless/static runtime)
- [x] 7.2 Create `deploy/serviceaccount.yaml` (ServiceAccount `lightspeed-agentic-alerts-adapter` in `openshift-lightspeed`)
- [x] 7.3 Create `deploy/rbac.yaml` (RoleBinding in `openshift-monitoring` for `monitoring-alertmanager-view`, ClusterRole + ClusterRoleBinding for Proposals create/list/get)
- [x] 7.4 Create `deploy/deployment.yaml` (single replica, probes on 8081, ServiceAccount reference)
