# ocpchart

Live ASCII Prometheus charts for OpenShift.

`ocpchart` queries Prometheus/Thanos on OpenShift clusters and renders live ASCII charts directly in the terminal. No database, no setup -- just point it at a cluster and start charting.

This whole project has been totally vibe coded using Cursor.

## Features

- **Interactive charts** -- run a PromQL range query, explore with pan/zoom/series navigation
- **Live mode** -- auto-refreshing charts with a rolling time window via `--refresh`

  <div align="center">
    <video src="https://github.com/greyerof/ocpchart/releases/download/v0.0.1/ocpchart-plot-live.mp4" width="90%" autoplay muted loop style="border: 2px solid #888;"></video>
  </div>

- **One-shot charts** -- static ASCII output via `--once`

  <div align="center">
    <video src="https://github.com/greyerof/ocpchart/releases/download/v0.0.1/ocpchart-plot-once.mp4" width="90%" autoplay muted loop style="border: 2px solid #888;"></video>
  </div>

- **Metric discovery** -- list all available Prometheus metrics, filter by regex

  <div align="center">
    <video src="https://github.com/greyerof/ocpchart/releases/download/v0.0.1/ocpchart-metrics-list.mp4" width="90%" autoplay muted loop style="border: 2px solid #888;"></video>
  </div>

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
# binary at ./ocpchart
```

## Quick start

```bash
# Discover metrics
ocpchart metrics list --kubeconfig ~/.kube/config
ocpchart metrics list --filter "cpu"

# Interactive chart with pan/zoom (default)
ocpchart plot 'rate(node_cpu_seconds_total{mode="idle"}[5m])' --since 1h

# Static one-shot chart
ocpchart plot 'node_memory_MemAvailable_bytes' --since 2h --once

# Live-refresh chart
ocpchart plot 'sum(up)' --since 30m --refresh 30s

# Live with faster refresh
ocpchart plot 'sum(up)' --since 1h --refresh 10s
```

## Authentication

### Kubeconfig (auto-discovery)

```bash
ocpchart plot 'up' --since 1h --kubeconfig ~/.kube/config
```

The kubeconfig user must have permission to create tokens for the `prometheus-k8s` service account in `openshift-monitoring`. ocpchart auto-discovers the Thanos Querier route and creates a short-lived bearer token.

If `--kubeconfig` is omitted, the `$KUBECONFIG` environment variable is used.

### Manual token + URL

```bash
ocpchart plot 'up' --since 1h \
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

### `ocpchart plot <promql>`

Run a range query and render an interactive ASCII chart. Charts are interactive by default with pan/zoom and series navigation.

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | *required* | How far back to query (e.g. `1h`, `30m`) |
| `--until` | now | End time (duration or RFC3339) |
| `--step` | auto | Query resolution step |
| `--once` | false | Print a static chart and exit (non-interactive) |
| `--refresh` | | Auto-refresh interval for live mode (e.g. `30s`, `1m`) |
| `--width` | terminal width | Chart width in columns (only with `--once`) |
| `--height` | 20 | Chart height in rows (only with `--once`) |

**Modes:**

- **Default (interactive):** full-screen chart with keyboard navigation. Data is fetched once.
- **`--once`:** prints a static chart and exits. If the query returns multiple series, you're prompted to pick one.
- **`--refresh`:** live auto-refreshing chart with a rolling time window. Mutually exclusive with `--once` and `--until`.

### Interactive controls

| Key | Action |
|-----|--------|
| Left / Right | Pan viewport |
| Up / Down | Zoom in / out |
| Space | Next time series |
| Backspace | Previous time series |
| g | Go-to series picker |
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
