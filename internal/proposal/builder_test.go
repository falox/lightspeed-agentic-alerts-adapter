package proposal

import (
	"testing"
	"time"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/constants"
)

func TestBuild_NamespacedAlert(t *testing.T) {
	alert := alertmanager.Alert{
		Labels: map[string]string{
			"alertname": "KubePodCrashLooping",
			"severity":  "warning",
			"namespace": "my-app",
		},
		Annotations: map[string]string{
			"summary":     "Pod is crash looping",
			"description": "Pod my-app/web-1 is restarting",
		},
		StartsAt:    time.Date(2026, 5, 26, 10, 15, 0, 0, time.UTC),
		Fingerprint: "a1b2c3d4e5f6",
	}

	p := Build(alert, "test request")

	if p.Namespace != "my-app" {
		t.Errorf("Namespace = %q, want %q", p.Namespace, "my-app")
	}
	if p.Name != "kubepodcrashlooping-my-app-a1b2c3d4" {
		t.Errorf("Name = %q, want %q", p.Name, "kubepodcrashlooping-my-app-a1b2c3d4")
	}
	if p.Labels[constants.LabelSource] != constants.SourceValue {
		t.Errorf("source label = %q, want %q", p.Labels[constants.LabelSource], constants.SourceValue)
	}
	if p.Labels[constants.LabelFingerprint] != "a1b2c3d4" {
		t.Errorf("fingerprint label = %q, want %q", p.Labels[constants.LabelFingerprint], "a1b2c3d4")
	}
	if p.Labels[constants.LabelAlertName] != "kubepodcrashlooping" {
		t.Errorf("alertname label = %q, want %q", p.Labels[constants.LabelAlertName], "kubepodcrashlooping")
	}
	if p.Labels[constants.LabelSeverity] != "warning" {
		t.Errorf("severity label = %q, want %q", p.Labels[constants.LabelSeverity], "warning")
	}
	if p.Annotations[constants.AnnotationStartsAt] != "2026-05-26T10:15:00Z" {
		t.Errorf("startsAt annotation = %q, want %q", p.Annotations[constants.AnnotationStartsAt], "2026-05-26T10:15:00Z")
	}
	if p.Annotations[constants.AnnotationSummary] != "Pod is crash looping" {
		t.Errorf("summary annotation = %q, want %q", p.Annotations[constants.AnnotationSummary], "Pod is crash looping")
	}
	if p.Spec.Request != "test request" {
		t.Errorf("Request = %q, want %q", p.Spec.Request, "test request")
	}
	if len(p.Spec.TargetNamespaces) != 1 || p.Spec.TargetNamespaces[0] != "my-app" {
		t.Errorf("TargetNamespaces = %v, want [my-app]", p.Spec.TargetNamespaces)
	}
	if p.Spec.Analysis.Agent != constants.DefaultAgent {
		t.Errorf("Analysis.Agent = %q, want %q", p.Spec.Analysis.Agent, constants.DefaultAgent)
	}
	if p.Spec.Execution.Agent != constants.DefaultAgent {
		t.Errorf("Execution.Agent = %q, want %q", p.Spec.Execution.Agent, constants.DefaultAgent)
	}
	if p.Spec.Verification.Agent != constants.DefaultAgent {
		t.Errorf("Verification.Agent = %q, want %q", p.Spec.Verification.Agent, constants.DefaultAgent)
	}
}

func TestBuild_ClusterScopedAlert(t *testing.T) {
	alert := alertmanager.Alert{
		Labels: map[string]string{
			"alertname": "EtcdHighFsyncDurations",
			"severity":  "critical",
		},
		Annotations: map[string]string{
			"summary": "Etcd fsync slow",
		},
		StartsAt:    time.Date(2026, 5, 26, 9, 30, 0, 0, time.UTC),
		Fingerprint: "f9e8d7c6b5a4",
	}

	p := Build(alert, "test request")

	if p.Namespace != constants.DefaultNamespace {
		t.Errorf("Namespace = %q, want %q", p.Namespace, constants.DefaultNamespace)
	}
	if p.Name != "etcdhighfsyncdurations-f9e8d7c6" {
		t.Errorf("Name = %q, want %q", p.Name, "etcdhighfsyncdurations-f9e8d7c6")
	}
	if p.Spec.TargetNamespaces != nil {
		t.Errorf("TargetNamespaces = %v, want nil", p.Spec.TargetNamespaces)
	}
}
