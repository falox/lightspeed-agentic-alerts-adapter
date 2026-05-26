package proposal

import (
	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/constants"
)

func Build(alert alertmanager.Alert, requestText string) *agenticv1alpha1.Proposal {
	ns := alert.Namespace()
	if ns == "" {
		ns = constants.DefaultNamespace
	}

	name := constants.ProposalName(alert.AlertName(), alert.Namespace(), alert.Fingerprint)
	fp := constants.FingerprintShort(alert.Fingerprint)

	p := &agenticv1alpha1.Proposal{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "agentic.openshift.io/v1alpha1",
			Kind:       "Proposal",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				constants.LabelSource:      constants.SourceValue,
				constants.LabelFingerprint: fp,
				constants.LabelAlertName:   sanitizeLabel(alert.AlertName()),
				constants.LabelSeverity:    sanitizeLabel(alert.Severity()),
			},
			Annotations: map[string]string{
				constants.AnnotationStartsAt: alert.StartsAt.Format("2006-01-02T15:04:05Z07:00"),
				constants.AnnotationSummary:  truncate(alert.Summary(), constants.MaxSummaryLen),
			},
		},
		Spec: agenticv1alpha1.ProposalSpec{
			Request: requestText,
			Analysis: agenticv1alpha1.ProposalStep{
				Agent: constants.DefaultAgent,
			},
			Execution: agenticv1alpha1.ProposalStep{
				Agent: constants.DefaultAgent,
			},
			Verification: agenticv1alpha1.ProposalStep{
				Agent: constants.DefaultAgent,
			},
		},
	}

	if alertNS := alert.Namespace(); alertNS != "" {
		p.Spec.TargetNamespaces = []string{alertNS}
	}

	return p
}

func sanitizeLabel(s string) string {
	s = constants.SanitizeDNS(s)
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
