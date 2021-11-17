// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license.

package query

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/timescale/promscale/pkg/telemetry"
	"github.com/timescale/promscale/pkg/util"
)

var (
	traceRequestsExec = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: util.PromNamespace,
		Subsystem: "trace",
		Name:      "query_requests_executed_total",
		Help:      "Total number of query requests successfully executed by /getTrace and /fetchTraces API.",
	})
	traceExecutionTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: util.PromNamespace,
		Subsystem: "trace",
		Name:      "query_execution_duration_seconds",
		Help:      "Time taken by a trace query for complete execution in /fetchTraces API.",
		Buckets:   append(prometheus.DefBuckets, []float64{30, 60, 120, 300, 600}...),
	})
	dependencyRequestsExec = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: util.PromNamespace,
		Subsystem: "trace",
		Name:      "dependency_requests_executed_total",
		Help:      "Total number of dependency requests successfully executed.",
	})
)

func registerMetricsForTelemetry(t telemetry.Telemetry) error {
	var err error
	if err = t.RegisterMetric("telemetry_traces_queries_executed", traceRequestsExec); err != nil {
		return fmt.Errorf("register 'telemetry_traces_queries_executed' metric for telemetry: %w", err)
	}
	if err = t.RegisterMetric("telemetry_traces_dependency_queries_executed", dependencyRequestsExec); err != nil {
		return fmt.Errorf("register 'telemetry_traces_dependency_queries_executed' metric for telemetry: %w", err)
	}
	return nil
}

func init() {
	prometheus.MustRegister(
		traceRequestsExec,
		traceExecutionTime,
		dependencyRequestsExec,
	)
}
