package constants

import (
	"strings"
	"testing"
)

func TestProposalName_Namespaced(t *testing.T) {
	got := ProposalName("KubePodCrashLooping", "production", "a1b2c3d4e5f6")
	want := "kubepodcrashlooping-production-a1b2c3d4"
	if got != want {
		t.Errorf("ProposalName() = %q, want %q", got, want)
	}
}

func TestProposalName_ClusterScoped(t *testing.T) {
	got := ProposalName("EtcdHighFsyncDurations", "", "f9e8d7c6b5a4")
	want := "etcdhighfsyncdurations-f9e8d7c6"
	if got != want {
		t.Errorf("ProposalName() = %q, want %q", got, want)
	}
}

func TestProposalName_Sanitization(t *testing.T) {
	got := ProposalName("My Alert!!", "ns--weird.stuff", "abcd1234")
	if strings.Contains(got, "!") {
		t.Error("name should not contain special characters")
	}
	if strings.Contains(got, "..") {
		t.Error("name should not contain dots")
	}
	if strings.Contains(got, "--") {
		t.Error("name should not contain consecutive hyphens")
	}
	if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
		t.Error("name should not start or end with hyphen")
	}
}

func TestProposalName_Truncation(t *testing.T) {
	longName := strings.Repeat("a", 300)
	got := ProposalName(longName, "ns", "abcd1234")
	if len(got) > 253 {
		t.Errorf("name length %d exceeds 253", len(got))
	}
	if strings.HasSuffix(got, "-") {
		t.Error("truncated name should not end with hyphen")
	}
}

func TestSanitizeDNS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"KubePodCrashLooping", "kubepodcrashlooping"},
		{"my--alert", "my-alert"},
		{"-leading-", "leading"},
		{"has spaces!", "has-spaces"},
	}
	for _, tt := range tests {
		got := SanitizeDNS(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeDNS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFingerprintShort(t *testing.T) {
	if got := FingerprintShort("a1b2c3d4e5f6"); got != "a1b2c3d4" {
		t.Errorf("FingerprintShort() = %q, want %q", got, "a1b2c3d4")
	}
	if got := FingerprintShort("short"); got != "short" {
		t.Errorf("FingerprintShort() = %q, want %q", got, "short")
	}
}
