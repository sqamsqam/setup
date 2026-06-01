package diagnostics

import (
	"strings"
	"testing"
)

func TestFormatReport(t *testing.T) {
	got := Format(Report{{Name: "Virtualization", Status: "ok", Detail: "lxc"}})
	if !strings.Contains(got, "Virtualization: ok") || !strings.Contains(got, "  lxc") {
		t.Fatalf("unexpected report: %q", got)
	}
}
