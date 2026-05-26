package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/constants"
)

type mockAlertFetcher struct {
	alerts []alertmanager.Alert
	err    error
}

func (m *mockAlertFetcher) FetchAlerts(_ context.Context) ([]alertmanager.Alert, error) {
	return m.alerts, m.err
}

func newFakeK8sClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = agenticv1alpha1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestRunCycle_CreatesProposalForFiringAlert(t *testing.T) {
	fetcher := &mockAlertFetcher{
		alerts: []alertmanager.Alert{
			{
				Labels: map[string]string{
					"alertname": "KubePodCrashLooping",
					"severity":  "warning",
					"namespace": "my-app",
				},
				Annotations: map[string]string{
					"summary":     "Pod is crash looping",
					"description": "Pod restarting",
				},
				StartsAt:    time.Now().Add(-10 * time.Minute),
				Fingerprint: "a1b2c3d4e5f6",
			},
		},
	}

	k8s := newFakeK8sClient()
	a := New(fetcher, k8s, testLogger())

	if err := a.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error: %v", err)
	}

	var proposals agenticv1alpha1.ProposalList
	if err := k8s.List(context.Background(), &proposals, client.InNamespace("my-app")); err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(proposals.Items) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals.Items))
	}

	p := proposals.Items[0]
	if p.Name != "kubepodcrashlooping-my-app-a1b2c3d4" {
		t.Errorf("name = %q, want %q", p.Name, "kubepodcrashlooping-my-app-a1b2c3d4")
	}
	if p.Labels[constants.LabelSource] != constants.SourceValue {
		t.Errorf("source label = %q", p.Labels[constants.LabelSource])
	}
}

func TestRunCycle_SkipsTooRecentAlert(t *testing.T) {
	fetcher := &mockAlertFetcher{
		alerts: []alertmanager.Alert{
			{
				Labels:      map[string]string{"alertname": "TestAlert", "namespace": "ns"},
				StartsAt:    time.Now().Add(-2 * time.Minute),
				Fingerprint: "abc12345dead",
			},
		},
	}

	k8s := newFakeK8sClient()
	a := New(fetcher, k8s, testLogger())

	if err := a.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error: %v", err)
	}

	var proposals agenticv1alpha1.ProposalList
	_ = k8s.List(context.Background(), &proposals)
	if len(proposals.Items) != 0 {
		t.Errorf("expected 0 proposals for too-recent alert, got %d", len(proposals.Items))
	}
}

func TestRunCycle_SkipsAlertWithActiveProposal(t *testing.T) {
	existing := &agenticv1alpha1.Proposal{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testalert-ns-abc12345",
			Namespace: "ns",
			Labels: map[string]string{
				constants.LabelSource:      constants.SourceValue,
				constants.LabelFingerprint: "abc12345",
			},
		},
		Status: agenticv1alpha1.ProposalStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Analyzed",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now().Add(-3 * time.Minute)),
				},
			},
		},
	}

	fetcher := &mockAlertFetcher{
		alerts: []alertmanager.Alert{
			{
				Labels:      map[string]string{"alertname": "TestAlert", "namespace": "ns"},
				StartsAt:    time.Now().Add(-10 * time.Minute),
				Fingerprint: "abc12345dead",
			},
		},
	}

	k8s := newFakeK8sClient(existing)
	a := New(fetcher, k8s, testLogger())

	if err := a.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error: %v", err)
	}

	var proposals agenticv1alpha1.ProposalList
	_ = k8s.List(context.Background(), &proposals)
	if len(proposals.Items) != 1 {
		t.Errorf("expected 1 proposal (existing only), got %d", len(proposals.Items))
	}
}

func TestRunCycle_AlertManagerError_SkipsCycle(t *testing.T) {
	fetcher := &mockAlertFetcher{err: fmt.Errorf("connection refused")}
	k8s := newFakeK8sClient()
	a := New(fetcher, k8s, testLogger())

	err := a.RunCycle(context.Background())
	if err == nil {
		t.Fatal("expected error from RunCycle when AlertManager is unreachable")
	}
}

func TestRunCycle_SkipsAlertWithNoAlertname(t *testing.T) {
	fetcher := &mockAlertFetcher{
		alerts: []alertmanager.Alert{
			{
				Labels:      map[string]string{"severity": "warning"},
				StartsAt:    time.Now().Add(-10 * time.Minute),
				Fingerprint: "abc12345dead",
			},
		},
	}

	k8s := newFakeK8sClient()
	a := New(fetcher, k8s, testLogger())

	if err := a.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error: %v", err)
	}

	var proposals agenticv1alpha1.ProposalList
	_ = k8s.List(context.Background(), &proposals)
	if len(proposals.Items) != 0 {
		t.Errorf("expected 0 proposals for alert without alertname, got %d", len(proposals.Items))
	}
}

func TestRunCycle_MultipleAlerts_MixedSkipAndCreate(t *testing.T) {
	fetcher := &mockAlertFetcher{
		alerts: []alertmanager.Alert{
			{
				Labels:      map[string]string{"alertname": "Alert1", "namespace": "ns1"},
				StartsAt:    time.Now().Add(-10 * time.Minute),
				Fingerprint: "1111111100000000",
			},
			{
				Labels:      map[string]string{"alertname": "Alert2", "namespace": "ns2"},
				StartsAt:    time.Now().Add(-1 * time.Minute),
				Fingerprint: "2222222200000000",
			},
			{
				Labels:      map[string]string{"alertname": "Alert3", "namespace": "ns3"},
				StartsAt:    time.Now().Add(-10 * time.Minute),
				Fingerprint: "3333333300000000",
			},
		},
	}

	k8s := newFakeK8sClient()
	a := New(fetcher, k8s, testLogger())

	if err := a.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle() error: %v", err)
	}

	var proposals agenticv1alpha1.ProposalList
	_ = k8s.List(context.Background(), &proposals)
	if len(proposals.Items) != 2 {
		t.Errorf("expected 2 proposals (Alert1+Alert3, Alert2 skipped for delay), got %d", len(proposals.Items))
	}
}
