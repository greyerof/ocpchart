package thanos

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/greyerof/ocpchart/internal/auth"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	rangeQueryTimeout  = 5 * time.Minute
	labelQueryTimeout  = 30 * time.Second
)

type Client struct {
	api promv1.API
}

// Series holds a single time series with its labels and data points.
type Series struct {
	Labels map[string]string
	Times  []time.Time
	Values []float64
}

// NewClient creates a Thanos/Prometheus API client from credentials.
func NewClient(creds *auth.Credentials) (*Client, error) {
	cfg := promapi.Config{
		Address: creds.ThanosURL,
		Client:  creds.HTTPClient(),
	}

	client, err := promapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating prometheus client: %w", err)
	}

	return &Client{api: promv1.NewAPI(client)}, nil
}

// MetricNames returns all metric names from the cluster, sorted.
func (c *Client) MetricNames(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, labelQueryTimeout)
	defer cancel()

	vals, _, err := c.api.LabelValues(ctx, "__name__", nil, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("fetching metric names: %w", err)
	}

	names := make([]string, len(vals))
	for i, v := range vals {
		names[i] = string(v)
	}

	sort.Strings(names)

	return names, nil
}

// RangeQuery executes a PromQL range query and returns the result as a slice of Series.
func (c *Client) RangeQuery(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]Series, error) {
	ctx, cancel := context.WithTimeout(ctx, rangeQueryTimeout)
	defer cancel()

	result, _, err := c.api.QueryRange(ctx, query, promv1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, fmt.Errorf("range query failed: %w", err)
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type %T, expected matrix", result)
	}

	series := make([]Series, 0, len(matrix))

	for _, stream := range matrix {
		labels := make(map[string]string, len(stream.Metric))
		for k, v := range stream.Metric {
			labels[string(k)] = string(v)
		}

		times := make([]time.Time, 0, len(stream.Values))
		values := make([]float64, 0, len(stream.Values))

		for _, sp := range stream.Values {
			v := float64(sp.Value)
			if v != v { // NaN check
				continue
			}

			times = append(times, sp.Timestamp.Time())
			values = append(values, v)
		}

		if len(values) == 0 {
			continue
		}

		series = append(series, Series{
			Labels: labels,
			Times:  times,
			Values: values,
		})
	}

	return series, nil
}

// LabelSetString returns a compact {k="v", ...} representation of a label set,
// excluding __name__ for brevity.
func LabelSetString(labels map[string]string) string {
	pairs := make([]string, 0, len(labels))

	for k, v := range labels {
		if k == "__name__" {
			continue
		}

		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, k, v))
	}

	sort.Strings(pairs)

	if len(pairs) == 0 {
		return "{}"
	}

	result := "{"
	for i, p := range pairs {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	result += "}"

	return result
}
