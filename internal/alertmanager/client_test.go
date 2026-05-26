package alertmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tokenDir := t.TempDir()
	tokenFile := filepath.Join(tokenDir, "token")
	if err := os.WriteFile(tokenFile, []byte("test-token"), 0600); err != nil {
		t.Fatal(err)
	}

	return &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
		tokenPath:  tokenFile,
	}
}

func TestFetchAlerts_ValidResponse(t *testing.T) {
	body := `[
		{
			"status": "firing",
			"labels": {"alertname": "KubePodCrashLooping", "severity": "warning", "namespace": "my-app"},
			"annotations": {"summary": "Pod is crash looping", "description": "Pod my-app/web-1 is restarting"},
			"startsAt": "2026-05-26T10:00:00Z",
			"fingerprint": "a1b2c3d4e5f6"
		},
		{
			"status": "firing",
			"labels": {"alertname": "EtcdHighFsyncDurations", "severity": "critical"},
			"annotations": {"summary": "Etcd fsync slow"},
			"startsAt": "2026-05-26T09:30:00Z",
			"fingerprint": "f9e8d7c6b5a4"
		}
	]`

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected bearer token")
		}
		if r.URL.Query().Get("active") != "true" {
			t.Error("expected active=true")
		}
		if r.URL.Query().Get("silenced") != "false" {
			t.Error("expected silenced=false")
		}
		if r.URL.Query().Get("inhibited") != "false" {
			t.Error("expected inhibited=false")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	})

	alerts, err := c.FetchAlerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}

	a := alerts[0]
	if a.AlertName() != "KubePodCrashLooping" {
		t.Errorf("expected alertname KubePodCrashLooping, got %s", a.AlertName())
	}
	if a.Severity() != "warning" {
		t.Errorf("expected severity warning, got %s", a.Severity())
	}
	if a.Namespace() != "my-app" {
		t.Errorf("expected namespace my-app, got %s", a.Namespace())
	}
	if a.Description() != "Pod my-app/web-1 is restarting" {
		t.Errorf("unexpected description: %s", a.Description())
	}
	if a.Fingerprint != "a1b2c3d4e5f6" {
		t.Errorf("expected fingerprint a1b2c3d4e5f6, got %s", a.Fingerprint)
	}

	b := alerts[1]
	if b.AlertName() != "EtcdHighFsyncDurations" {
		t.Errorf("expected alertname EtcdHighFsyncDurations, got %s", b.AlertName())
	}
	if b.Namespace() != "" {
		t.Errorf("expected empty namespace for cluster-scoped alert, got %s", b.Namespace())
	}
}

func TestFetchAlerts_EmptyResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	alerts, err := c.FetchAlerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(alerts))
	}
}

func TestFetchAlerts_NonOKStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.FetchAlerts(context.Background())
	if err == nil {
		t.Fatal("expected error for non-OK status")
	}
}

func TestFetchAlerts_InvalidJSON(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	})

	_, err := c.FetchAlerts(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
