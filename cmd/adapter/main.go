package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/adapter"
	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
)

const (
	PollInterval   = 30 * time.Second
	AlertManagerURL = "https://alertmanager-main.openshift-monitoring.svc:9094"
	HealthAddr     = ":8081"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	scheme := runtime.NewScheme()
	utilruntime.Must(agenticv1alpha1.AddToScheme(scheme))

	cfg, err := config.GetConfig()
	if err != nil {
		slog.Error("failed to get in-cluster config", "error", err)
		os.Exit(1)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		slog.Error("failed to create kubernetes client", "error", err)
		os.Exit(1)
	}

	amClient, err := alertmanager.NewClient(AlertManagerURL)
	if err != nil {
		slog.Error("failed to create alertmanager client", "error", err)
		os.Exit(1)
	}

	a := adapter.New(amClient, k8sClient, logger)

	var ready atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready.Load() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	server := &http.Server{Addr: HealthAddr, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server failed", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	slog.Info("starting alerts adapter", "pollInterval", PollInterval, "alertManagerURL", AlertManagerURL)

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	runCycle := func() {
		if err := a.RunCycle(ctx); err != nil {
			slog.Error("poll cycle failed", "error", err)
		} else {
			ready.Store(true)
		}
	}

	runCycle()
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			server.Shutdown(context.Background())
			return
		case <-ticker.C:
			runCycle()
		}
	}
}
