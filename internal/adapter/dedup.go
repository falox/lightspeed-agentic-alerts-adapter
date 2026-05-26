package adapter

import (
	"time"

	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/constants"
)

const (
	InitialDelay   = 5 * time.Minute
	CooldownWindow = 1 * time.Hour
)

type SkipReason string

const (
	SkipNone           SkipReason = ""
	SkipInitialDelay   SkipReason = "initial-delay"
	SkipActiveProposal SkipReason = "active-proposal"
	SkipCooldown       SkipReason = "cooldown"
)

func ShouldSkip(alert alertmanager.Alert, proposals []agenticv1alpha1.Proposal, now time.Time) SkipReason {
	if now.Sub(alert.StartsAt) < InitialDelay {
		return SkipInitialDelay
	}

	fp := constants.FingerprintShort(alert.Fingerprint)
	for i := range proposals {
		p := &proposals[i]
		pFP := p.Labels[constants.LabelFingerprint]
		if pFP != fp {
			continue
		}

		phase := agenticv1alpha1.DerivePhase(p.Status.Conditions)
		if !isTerminal(phase) {
			return SkipActiveProposal
		}

		terminalTime := terminalTimestamp(p)
		if !terminalTime.IsZero() && now.Sub(terminalTime) < CooldownWindow {
			return SkipCooldown
		}
	}

	return SkipNone
}

func isTerminal(phase agenticv1alpha1.ProposalPhase) bool {
	switch phase {
	case agenticv1alpha1.ProposalPhaseCompleted,
		agenticv1alpha1.ProposalPhaseFailed,
		agenticv1alpha1.ProposalPhaseDenied,
		agenticv1alpha1.ProposalPhaseEscalated:
		return true
	}
	return false
}

func terminalTimestamp(p *agenticv1alpha1.Proposal) time.Time {
	var latest time.Time
	for _, c := range p.Status.Conditions {
		if c.LastTransitionTime.Time.After(latest) {
			latest = c.LastTransitionTime.Time
		}
	}
	return latest
}
