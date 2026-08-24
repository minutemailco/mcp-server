package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestToolCallsTotalNoPrecreatedChildren(t *testing.T) {
	// Children must NOT be pre-created: series appear on first call only.
	if n := testutil.CollectAndCount(ToolCallsTotal); n != 0 {
		t.Fatalf("expected 0 pre-created children, got %d", n)
	}
}

func TestToolCallsTotalIncrements(t *testing.T) {
	ToolCallsTotal.WithLabelValues("mailboxes.list", ResultSuccess).Inc()
	ToolCallsTotal.WithLabelValues("mailboxes.list", ResultSuccess).Inc()
	ToolCallsTotal.WithLabelValues("mailboxes.create", ResultError).Inc()

	if v := testutil.ToFloat64(ToolCallsTotal.WithLabelValues("mailboxes.list", ResultSuccess)); v != 2 {
		t.Fatalf("mailboxes.list success = %v, want 2", v)
	}
	if v := testutil.ToFloat64(ToolCallsTotal.WithLabelValues("mailboxes.create", ResultError)); v != 1 {
		t.Fatalf("mailboxes.create error = %v, want 1", v)
	}
}
