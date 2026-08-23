# mcp-server Metrics

Prometheus metrics for the MinuteMail MCP server.

## Endpoint

`GET /metrics` — Prometheus text format, no authentication. Served on the same
port as `/mcp` and `/health` (default `PORT=8080`; Service port 80).

## Exposed metrics

The endpoint uses `promhttp.Handler()` from `prometheus/client_golang` on the
default registry, so it exports the standard Go and process runtime metrics,
e.g.:

- `go_goroutines`, `go_memstats_*`, `go_gc_duration_seconds` — Go runtime
- `process_cpu_seconds_total`, `process_resident_memory_bytes` — process

## Business Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `mm_mcp_tool_calls_total` | Counter | `tool`, `result` | MCP tool calls dispatched via `tools/call`. `tool` is the registry-resolved tool name (e.g. `mm_list_mailboxes`); `result="success"` for a completed call with a non-error result, `result="error"` when the call fails: invalid arguments, missing bearer, handler/transport failure, or an `isError` result (non-2xx gateway response such as 401/403/429/5xx). Label children are NOT pre-created — with 36 tools the series appear on first call instead of at process start (same approach as `mm_webhook_events_total` in stripe-webhook). |

Instrumentation lives at the single dispatch point `handleToolsCall` in
`internal/mcp/server.go`; the metric is defined in `internal/metrics/metrics.go`.
Calls that fail before the tool name resolves (malformed params, unknown tool)
are not counted, keeping the `tool` label limited to registered tool names.

## Useful Queries

```promql
# Tool call rate by result
sum by (result) (rate(mm_mcp_tool_calls_total[5m]))

# Error ratio
sum(rate(mm_mcp_tool_calls_total{result="error"}[5m]))
  / sum(rate(mm_mcp_tool_calls_total[5m]))

# Top tools by call volume
topk(10, sum by (tool) (rate(mm_mcp_tool_calls_total[5m])))
```

## Scrape configuration

The Helm chart (`chart/`) adds pod annotations (default ON, controlled by
`metrics.annotations.enabled` in `values.yaml`):

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8080"
prometheus.io/path: "/metrics"
```

Any Prometheus with Kubernetes pod discovery picking up `prometheus.io/*`
annotations will scrape this service automatically. Equivalent manual scrape
config:

```yaml
scrape_configs:
  - job_name: 'mcp-server'
    metrics_path: /metrics
    static_configs:
      - targets: ['mcp-server:80']
```
