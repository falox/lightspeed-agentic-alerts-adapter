package alertmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	tokenPath  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	caPath     = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
	alertsPath = "/api/v2/alerts"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	tokenPath  string
}

func NewClient(baseURL string) (*Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	caData, err := os.ReadFile(caPath)
	if err == nil {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caData)
		transport.TLSClientConfig = &tls.Config{
			RootCAs: pool,
		}
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
		tokenPath: tokenPath,
	}, nil
}

func (c *Client) FetchAlerts(ctx context.Context) ([]Alert, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+alertsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	q := req.URL.Query()
	q.Set("active", "true")
	q.Set("silenced", "false")
	q.Set("inhibited", "false")
	req.URL.RawQuery = q.Encode()

	token, err := os.ReadFile(c.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("reading service account token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching alerts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from AlertManager", resp.StatusCode)
	}

	var alerts []Alert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, fmt.Errorf("decoding alerts: %w", err)
	}

	return alerts, nil
}
