# ocpchart

Live ASCII Prometheus charts for OpenShift.

`ocpchart` queries Prometheus/Thanos on OpenShift clusters and renders live ASCII charts directly in the terminal. No database, no setup -- just point it at a cluster and start charting.

## Features

- **Metric discovery** -- list all available Prometheus metrics, filter by regex
- **One-shot charts** -- run a PromQL range query, get an ASCII chart
- **Live mode** -- auto-refreshing charts with a rolling time window
- **Interactive navigation** -- pan, zoom, and switch between time series
- **Two auth modes** -- kubeconfig auto-discovery or manual token + URL

## Installation

### From source

```bash
go install github.com/greyerof/ocpchart/cmd/ocpchart@latest
```

### Build locally

```bash
git clone https://github.com/greyerof/ocpchart.git
cd ocpchart
make build
# binary at bin/ocpchart
```

## Quick start

```bash
# Discover metrics
ocpchart metrics list --kubeconfig ~/.kube/config
ocpchart metrics list --filter "cpu"

# One-shot chart (last hour, auto step)
ocpchart query 'rate(node_cpu_seconds_total{mode="idle"}[5m])' --since 1h

# Interactive chart with pan/zoom
ocpchart query 'node_memory_MemAvailable_bytes' --since 2h -i

# Live-refresh chart (updates every 30s)
ocpchart live 'rate(node_cpu_seconds_total{mode="idle"}[5m])' --since 30m

# Live with custom refresh interval
ocpchart live 'sum(up)' --since 1h --refresh 10s
```

## Authentication

### Kubeconfig (auto-discovery)

```bash
ocpchart query 'up' --since 1h --kubeconfig ~/.kube/config
```

The kubeconfig user must have permission to create tokens for the `prometheus-k8s` service account in `openshift-monitoring`. ocpchart auto-discovers the Thanos Querier route and creates a short-lived bearer token.

If `--kubeconfig` is omitted, the `$KUBECONFIG` environment variable is used.

### Manual token + URL

```bash
ocpchart query 'up' --since 1h \
  --thanos-url https://thanos-querier.apps.cluster.example.com \
  --token <bearer-token>
```

Both `--token` and `--thanos-url` must be provided together.

### TLS

Use `--insecure-tls` to skip certificate verification (e.g., self-signed clusters).

## Commands

### `ocpchart metrics list`

List available Prometheus metric names.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--filter` | `-f` | | Regex to filter metric names |

### `ocpchart query <promql>`

Run a range query and render a chart.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--since` | | *required* | How far back to query (e.g. `1h`, `30m`) |
| `--until` | | now | End time (duration or RFC3339) |
| `--step` | | auto | Query resolution step |
| `--width` | | terminal width | Chart width in columns |
| `--height` | | 20 | Chart height in rows |
| `--interactive` | `-i` | false | Interactive pan/zoom mode |

**Multi-series:** if the query returns multiple series, you're prompted to pick one. In interactive mode (`-i`), use Space/Backspace to cycle through them.

### `ocpchart live <promql>`

Auto-refreshing chart with a rolling time window.

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | *required* | Rolling window size |
| `--step` | auto | Query resolution step |
| `--refresh` | `30s` | How often to re-query |

Live mode is always interactive (full-screen).

### Interactive controls

| Key | Action |
|-----|--------|
| Left / Right | Pan viewport |
| Up / Down | Zoom in / out |
| Space | Next time series |
| Backspace | Previous time series |
| q / Ctrl+C | Quit |

## License

Apache License 2.0
