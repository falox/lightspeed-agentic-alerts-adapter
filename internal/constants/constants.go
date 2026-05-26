package constants

import (
	"regexp"
	"strings"
)

const (
	LabelSource      = "agentic.openshift.io/source"
	LabelFingerprint = "agentic.openshift.io/alert-fingerprint"
	LabelAlertName   = "agentic.openshift.io/alert-name"
	LabelSeverity    = "agentic.openshift.io/alert-severity"

	AnnotationStartsAt = "agentic.openshift.io/alert-starts-at"
	AnnotationSummary  = "agentic.openshift.io/alert-summary"

	SourceValue      = "alertmanager"
	DefaultNamespace = "openshift-lightspeed"
	DefaultAgent     = "default"
	MaxSummaryLen    = 256
)

var nonAlphanumericHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
var multipleHyphens = regexp.MustCompile(`-{2,}`)

func SanitizeDNS(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumericHyphen.ReplaceAllString(s, "-")
	s = multipleHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func FingerprintShort(fingerprint string) string {
	if len(fingerprint) > 8 {
		return fingerprint[:8]
	}
	return fingerprint
}

func ProposalName(alertName, namespace, fingerprint string) string {
	an := SanitizeDNS(alertName)
	ns := SanitizeDNS(namespace)
	fp := SanitizeDNS(FingerprintShort(fingerprint))

	var name string
	if ns == "" {
		name = an + "-" + fp
	} else {
		name = an + "-" + ns + "-" + fp
	}

	if len(name) > 253 {
		name = name[:253]
	}
	name = strings.TrimRight(name, "-")
	return name
}
