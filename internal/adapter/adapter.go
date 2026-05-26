package adapter

import (
	"context"
	"log/slog"
	"time"

	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/constants"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/proposal"
)

type AlertFetcher interface {
	FetchAlerts(ctx context.Context) ([]alertmanager.Alert, error)
}

type Adapter struct {
	amClient  AlertFetcher
	k8sClient client.Client
	logger    *slog.Logger
}

func New(amClient AlertFetcher, k8sClient client.Client, logger *slog.Logger) *Adapter {
	return &Adapter{
		amClient:  amClient,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

func (a *Adapter) RunCycle(ctx context.Context) error {
	now := time.Now()

	alerts, err := a.amClient.FetchAlerts(ctx)
	if err != nil {
		return err
	}

	var proposalList agenticv1alpha1.ProposalList
	if err := a.k8sClient.List(ctx, &proposalList, client.MatchingLabels{
		constants.LabelSource: constants.SourceValue,
	}); err != nil {
		return err
	}

	a.logger.Info("poll cycle", "alerts", len(alerts), "existingProposals", len(proposalList.Items))

	var created int
	for _, alert := range alerts {
		if alert.AlertName() == "" {
			a.logger.Debug("skipping alert with no alertname", "fingerprint", alert.Fingerprint)
			continue
		}

		reason := ShouldSkip(alert, proposalList.Items, now)
		if reason != SkipNone {
			a.logger.Debug("skipping alert",
				"alertname", alert.AlertName(),
				"fingerprint", constants.FingerprintShort(alert.Fingerprint),
				"reason", string(reason),
			)
			continue
		}

		if err := a.createProposal(ctx, alert); err != nil {
			a.logger.Error("failed to create proposal",
				"alertname", alert.AlertName(),
				"namespace", alert.Namespace(),
				"fingerprint", constants.FingerprintShort(alert.Fingerprint),
				"error", err,
			)
			continue
		}
		created++
	}

	if created > 0 {
		a.logger.Info("poll cycle complete", "created", created)
	}
	return nil
}

func (a *Adapter) createProposal(ctx context.Context, alert alertmanager.Alert) error {
	requestText, err := RenderRequest(alert)
	if err != nil {
		return err
	}

	p := proposal.Build(alert, requestText)

	if err := a.k8sClient.Create(ctx, p); err != nil {
		if errors.IsAlreadyExists(err) {
			a.logger.Debug("proposal already exists",
				"name", p.Name,
				"namespace", p.Namespace,
			)
			return nil
		}
		return err
	}

	a.logger.Info("created proposal",
		"name", p.Name,
		"namespace", p.Namespace,
		"alertname", alert.AlertName(),
	)
	return nil
}
