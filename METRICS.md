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

There are no mcp-server-specific business metrics yet; the default registry
metrics are sufficient to monitor goroutine counts, memory, and restarts.

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
