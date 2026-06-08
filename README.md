# ocpchart

Live ASCII Prometheus charts for OpenShift.

`ocpchart` queries Prometheus/Thanos on OpenShift clusters and renders live ASCII charts directly in the terminal. No database, no setup -- just point it at a cluster and start charting.

This whole project has been totally vibe coded using Cursor.

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

## Annex: Internals

### Thanos route discovery path

When using kubeconfig auth mode, `ocpchart` does not hardcode a cluster-specific hostname.
It calls the OpenShift Route API and reads:

- `GET /apis/route.openshift.io/v1/namespaces/openshift-monitoring/routes/thanos-querier`

The response field `spec.host` is used to build the final endpoint as:

- `https://<spec.host>`

That URL is then passed to the Prometheus v1 API client for all metric and range queries.

### Service account token creation

`ocpchart` requests a token from Kubernetes via the TokenRequest API for:

- namespace: `openshift-monitoring`
- service account: `prometheus-k8s`

The token is intentionally short-lived (`10m`) so leaked credentials have a smaller blast radius and users naturally re-auth with fresh credentials on subsequent command runs.

In practice this means:

- each command run in kubeconfig mode gets a fresh bearer token
- long-lived static secrets are avoided by default
- manual mode (`--token` + `--thanos-url`) remains available when token lifecycle is managed externally

### Why short expiration matters in this tool

`ocpchart` is a CLI meant for ad-hoc, terminal-driven observability sessions. A short expiry fits this workflow because:

- sessions are usually brief (query or live window while debugging)
- token reuse across unrelated sessions is discouraged
- accidental token exposure in shell history or logs has limited lifetime impact

### Query timing behavior

Some timing internals worth knowing:

- `--until` accepts either a Go duration (interpreted as `now - duration`) or an RFC3339 timestamp
- omitted `--until` defaults to `now`
- range query timeout is 5 minutes
- metric name discovery timeout is 30 seconds

### Step auto-calculation heuristics

If `--step` is omitted:

- `ocpchart` estimates a datapoint count from terminal width (`width - 10`)
- clamps the minimum number of points to 20
- clamps minimum step to `15s`

This keeps charts readable on narrow terminals while avoiding very high-frequency queries by default.
