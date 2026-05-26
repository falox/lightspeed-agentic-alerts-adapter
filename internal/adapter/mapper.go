package adapter

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/openshift/lightspeed-agentic-alerts-adapter/internal/alertmanager"
)

var requestTemplate = template.Must(template.New("request").Parse(
	`Alert: {{.AlertName}} (severity: {{.Severity}})
Namespace: {{.Namespace}}

{{.Description}}

Investigate the root cause and propose a remediation.

Alert labels:
{{- range $k, $v := .Labels}}
- {{$k}}: {{$v}}
{{- end}}
`))

type templateData struct {
	AlertName   string
	Severity    string
	Namespace   string
	Description string
	Labels      map[string]string
}

func RenderRequest(alert alertmanager.Alert) (string, error) {
	data := templateData{
		AlertName:   alert.AlertName(),
		Severity:    alert.Severity(),
		Namespace:   alert.Namespace(),
		Description: alert.Description(),
		Labels:      alert.Labels,
	}

	var buf bytes.Buffer
	if err := requestTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering request template: %w", err)
	}
	return buf.String(), nil
}
