package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
)

func TestRenderRequest(t *testing.T) {
	alert := alertmanager.Alert{
		Labels: map[string]string{
			"alertname": "KubePodCrashLooping",
			"severity":  "warning",
			"namespace": "my-app",
		},
		Annotations: map[string]string{
			"description": "Pod my-app/web-1 is restarting",
		},
		StartsAt:    time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC),
		Fingerprint: "a1b2c3d4e5f6",
	}

	result, err := RenderRequest(alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "KubePodCrashLooping") {
		t.Error("result should contain alert name")
	}
	if !strings.Contains(result, "warning") {
		t.Error("result should contain severity")
	}
	if !strings.Contains(result, "my-app") {
		t.Error("result should contain namespace")
	}
	if !strings.Contains(result, "Pod my-app/web-1 is restarting") {
		t.Error("result should contain description")
	}
	if !strings.Contains(result, "Investigate the root cause") {
		t.Error("result should contain investigation instruction")
	}
	if !strings.Contains(result, "alertname: KubePodCrashLooping") {
		t.Error("result should contain labels")
	}
}
