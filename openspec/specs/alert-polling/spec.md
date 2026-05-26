# Alert Polling

## Purpose

Defines how the adapter polls AlertManager for firing alerts, including authentication, error handling, and lifecycle management.

## Requirements

### Requirement: Poll AlertManager for firing alerts
The adapter SHALL poll AlertManager's v2 API at a fixed interval of 30 seconds, requesting only active, non-silenced, non-inhibited alerts via `GET /api/v2/alerts?active=true&silenced=false&inhibited=false`.

#### Scenario: Successful poll cycle
- **WHEN** the poll interval elapses
- **THEN** the adapter sends a GET request to `https://alertmanager-main.openshift-monitoring.svc:9094/api/v2/alerts?active=true&silenced=false&inhibited=false`
- **AND** parses the JSON response into a list of alerts with their labels, annotations, startsAt, and fingerprint

#### Scenario: AlertManager is unreachable
- **WHEN** the adapter cannot connect to AlertManager or receives a non-2xx response
- **THEN** the adapter logs an error and skips the entire poll cycle
- **AND** retries on the next poll interval

#### Scenario: Invalid alert in response
- **WHEN** an individual alert in the response has missing or invalid data (e.g., no alertname label)
- **THEN** the adapter logs and skips that alert
- **AND** continues processing the remaining alerts in the same response

### Requirement: Authenticate to AlertManager using ServiceAccount token
The adapter SHALL authenticate to AlertManager using a Bearer token from the pod's auto-mounted ServiceAccount token at `/var/run/secrets/kubernetes.io/serviceaccount/token`. TLS SHALL be verified against the cluster CA bundle.

#### Scenario: Successful authentication
- **WHEN** the adapter sends a request to AlertManager
- **THEN** it includes a Bearer token from `/var/run/secrets/kubernetes.io/serviceaccount/token`
- **AND** verifies the TLS certificate against `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`

### Requirement: Continuous polling with graceful shutdown
The adapter SHALL run the poll loop continuously until it receives a termination signal (SIGTERM/SIGINT), at which point it SHALL complete the current cycle and shut down gracefully.

#### Scenario: Graceful shutdown
- **WHEN** the adapter receives SIGTERM
- **THEN** it completes the in-progress poll cycle (if any)
- **AND** exits cleanly
