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
	ToolCallsTotal.WithLabelValues("mm_list_mailboxes", ResultSuccess).Inc()
	ToolCallsTotal.WithLabelValues("mm_list_mailboxes", ResultSuccess).Inc()
	ToolCallsTotal.WithLabelValues("mm_create_mailbox", ResultError).Inc()

	if v := testutil.ToFloat64(ToolCallsTotal.WithLabelValues("mm_list_mailboxes", ResultSuccess)); v != 2 {
		t.Fatalf("mm_list_mailboxes success = %v, want 2", v)
	}
	if v := testutil.ToFloat64(ToolCallsTotal.WithLabelValues("mm_create_mailbox", ResultError)); v != 1 {
		t.Fatalf("mm_create_mailbox error = %v, want 1", v)
	}
}
