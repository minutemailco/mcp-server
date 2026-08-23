// Package metrics defines the business metrics exposed on /metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Result label values for ToolCallsTotal.
const (
	ResultSuccess = "success"
	ResultError   = "error"
)

// ToolCallsTotal counts MCP tool calls by tool name and outcome. Label
// children are intentionally NOT pre-created: the tool label spans the 36
// registered tools, so series appear on first call rather than at process
// start (same approach as mm_webhook_events_total in stripe-webhook).
var ToolCallsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "mm_mcp_tool_calls_total",
		Help: "Total number of MCP tool calls by tool and result.",
	},
	[]string{"tool", "result"},
)
