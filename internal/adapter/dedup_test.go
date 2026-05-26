package adapter

import (
	"testing"
	"time"

	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/constants"
)

func TestShouldSkip_InitialDelay(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		startsAt time.Time
		want     SkipReason
	}{
		{"3 minutes ago - too recent", now.Add(-3 * time.Minute), SkipInitialDelay},
		{"6 minutes ago - past delay", now.Add(-6 * time.Minute), SkipNone},
		{"exactly 5 minutes ago - boundary", now.Add(-5 * time.Minute), SkipNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := alertmanager.Alert{
				StartsAt:    tt.startsAt,
				Fingerprint: "abc12345dead",
				Labels:      map[string]string{"alertname": "TestAlert"},
			}
			got := ShouldSkip(alert, nil, now)
			if got != tt.want {
				t.Errorf("ShouldSkip() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldSkip_ActiveProposal(t *testing.T) {
	now := time.Now()
	alert := alertmanager.Alert{
		StartsAt:    now.Add(-10 * time.Minute),
		Fingerprint: "abc12345dead",
		Labels:      map[string]string{"alertname": "TestAlert"},
	}

	proposals := []agenticv1alpha1.Proposal{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					constants.LabelFingerprint: "abc12345",
				},
			},
			Status: agenticv1alpha1.ProposalStatus{
				Conditions: []metav1.Condition{
					{
						Type:               "Analyzed",
						Status:             metav1.ConditionUnknown,
						LastTransitionTime: metav1.NewTime(now.Add(-5 * time.Minute)),
					},
				},
			},
		},
	}

	got := ShouldSkip(alert, proposals, now)
	if got != SkipActiveProposal {
		t.Errorf("ShouldSkip() = %q, want %q", got, SkipActiveProposal)
	}
}

func TestShouldSkip_CooldownWindow(t *testing.T) {
	now := time.Now()
	alert := alertmanager.Alert{
		StartsAt:    now.Add(-10 * time.Minute),
		Fingerprint: "abc12345dead",
		Labels:      map[string]string{"alertname": "TestAlert"},
	}

	t.Run("within cooldown", func(t *testing.T) {
		proposals := []agenticv1alpha1.Proposal{
			{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.LabelFingerprint: "abc12345",
					},
				},
				Status: agenticv1alpha1.ProposalStatus{
					Conditions: []metav1.Condition{
						{
							Type:               "Verified",
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(now.Add(-30 * time.Minute)),
						},
					},
				},
			},
		}
		got := ShouldSkip(alert, proposals, now)
		if got != SkipCooldown {
			t.Errorf("ShouldSkip() = %q, want %q", got, SkipCooldown)
		}
	})

	t.Run("outside cooldown", func(t *testing.T) {
		proposals := []agenticv1alpha1.Proposal{
			{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.LabelFingerprint: "abc12345",
					},
				},
				Status: agenticv1alpha1.ProposalStatus{
					Conditions: []metav1.Condition{
						{
							Type:               "Verified",
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(now.Add(-2 * time.Hour)),
						},
					},
				},
			},
		}
		got := ShouldSkip(alert, proposals, now)
		if got != SkipNone {
			t.Errorf("ShouldSkip() = %q, want %q", got, SkipNone)
		}
	})
}

func TestShouldSkip_NoMatchingFingerprint(t *testing.T) {
	now := time.Now()
	alert := alertmanager.Alert{
		StartsAt:    now.Add(-10 * time.Minute),
		Fingerprint: "abc12345dead",
		Labels:      map[string]string{"alertname": "TestAlert"},
	}

	proposals := []agenticv1alpha1.Proposal{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					constants.LabelFingerprint: "ffffffff",
				},
			},
		},
	}

	got := ShouldSkip(alert, proposals, now)
	if got != SkipNone {
		t.Errorf("ShouldSkip() = %q, want %q", got, SkipNone)
	}
}
