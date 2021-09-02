package integration

import (
	"time"

	"github.com/timescale/promscale/pkg/tests/common"
)

type testQuery struct {
	name       string
	expression string
	start, end int64
}

var startTimeOffset = common.StartTime + 24*30*time.Hour.Milliseconds() // start time + 1 month.

var benchmarkableInstantQueries = []testQuery{
	// Real-world inspired PromQL queries.
	// References:
	// 1. https://www.robustperception.io/common-query-patterns-in-promql
	// 2. https://github.com/infinityworks/prometheus-example-queries
	{
		name:       "vs: simple gauge 1",
		expression: "one_gauge_metric",
		start:      startTimeOffset,
		end:        startTimeOffset,
	},
	{
		name:       "vs: simple gauge 2",
		expression: "two_gauge_metric",
		start:      startTimeOffset,
		end:        startTimeOffset,
	},
	{
		name:       "utilization percentage",
		expression: "100 * (1 - avg by(instance)(irate(one_gauge_metric{foo='bar'}[5m])))",
		start:      startTimeOffset,
		end:        startTimeOffset,
	},
	{
		name: "event occurange percentage (like rate of errors)",
		expression: `
	rate(one_counter_total[5m]) * 50
> on(job, instance)
	rate(two_counter_total[5m])`,
		start: startTimeOffset,
		end:   startTimeOffset,
	},
	{
		name:       "percentile calculation 1",
		expression: `histogram_quantile(0.9, one_histogram_bucket)`,
		start:      startTimeOffset,
		end:        startTimeOffset,
	},
	{
		name: "percentile calculation 2",
		expression: `
	histogram_quantile(0.9, rate(one_histogram_bucket{job="benchmark"}[10m])) > 0.05
and
	rate(one_histogram_count[10m]) > 1`,
		start: startTimeOffset,
		end:   startTimeOffset,
	},
}

var realInstantQueries = []testQuery{
	{
		name:       "simple gauge: percent of request in last 30 days answered successfully",
		expression: `apiserver_request:availability30d{verb="all", cluster="$cluster"}`,
	}, {
		name:       "binary: error left at 0.990% availability",
		expression: `100 * (apiserver_request:availability30d{verb="all", cluster="$cluster"} - 0.990000)`,
	}, {
		name:       "sum: total read requests per second to apiserver",
		expression: `sum by (code) (code_resource:apiserver_request_total:rate5m{verb=\"read\", cluster=\"$cluster\"})`,
	}, {
		name:       "sum + binary: percent of read requests per second returned with errors",
		expression: `sum by (resource) (code_resource:apiserver_request_total:rate5m{verb=\"read\",code=~\"5..\", cluster=\"$cluster\"}) / sum by (resource) (code_resource:apiserver_request_total:rate5m{verb=\"read\", cluster=\"$cluster\"})`,
	}, {
		name:       "histogram quantile: 99th percentile for reading a resource in seconds",
		expression: `cluster_quantile:apiserver_request_duration_seconds:histogram_quantile{verb=\"read\", cluster=\"$cluster\"}`,
	}, {
		name:       "sum + binary: percent write request per second returned with errors",
		expression: `sum by (resource) (code_resource:apiserver_request_total:rate5m{verb=\"write\",code=~\"5..\", cluster=\"$cluster\"}) / sum by (resource) (code_resource:apiserver_request_total:rate5m{verb=\"write\", cluster=\"$cluster\"})`,
	}, {
		name:       "sum + rate: rate work queue addition",
		expression: `sum(rate(workqueue_adds_total{job=\"kube-apiserver\", instance=~\"$instance\", cluster=\"$cluster\"}[5m])) by (instance, name)`,
	}, {
		name:       "histogram + sum + rate: 0.99 quantile of work queue bucket duration",
		expression: `histogram_quantile(0.99, sum(rate(workqueue_queue_duration_seconds_bucket{job=\"kube-apiserver\", instance=~\"$instance\", cluster=\"$cluster\"}[5m])) by (instance, name, le))`,
	}, {
		name:       "rate: process cpu seconds",
		expression: `rate(process_cpu_seconds_total{job=\"kube-apiserver\",instance=~\"$instance\", cluster=\"$cluster\"}[5m])`,
	}, {
		name:       "call: fetching label values of apiserver request",
		expression: `label_values(apiserver_request_total{job=\"kube-apiserver\", cluster=\"$cluster\"}, instance)`,
	}, {
		name:       "sort + sum + irate (regex all): total container network bytes",
		expression: `sort_desc(sum(irate(container_network_receive_bytes_total{cluster=\"$cluster\",namespace=~\".+\"}[$interval:$resolution])) by (namespace))`,
	}, {
		name:       "sort + avg + irate (regex all): total container network bytes",
		expression: `sort_desc(avg(irate(container_network_receive_bytes_total{cluster=\"$cluster\",namespace=~\".+\"}[$interval:$resolution])) by (namespace))`,
	}, {
		name:       "sort + sum + rate + binary: rate of retrans segs by rate of out segs",
		expression: `sort_desc(sum(rate(node_netstat_Tcp_RetransSegs{cluster=\"$cluster\"}[$interval:$resolution]) / rate(node_netstat_Tcp_OutSegs{cluster=\"$cluster\"}[$interval:$resolution])) by (instance))`,
	}, {
		name:       "sum + rate: work queues adds total",
		expression: `sum(rate(workqueue_adds_total{cluster=\"$cluster\", job=\"kube-controller-manager\", instance=~\"$instance\"}[5m])) by (cluster, instance, name)`,
	}, {
		name:       "histogram quantile + sum + rate: 99th percentile of work queue duration",
		expression: `histogram_quantile(0.99, sum(rate(workqueue_queue_duration_seconds_bucket{cluster=\"$cluster\", job=\"kube-controller-manager\", instance=~\"$instance\"}[5m])) by (cluster, instance, name, le))`,
	}, {
		name:       "sum + rate: rest client request 2xx",
		expression: `sum(rate(rest_client_requests_total{job=\"kube-controller-manager\", instance=~\"$instance\",code=~\"2..\"}[5m]))`,
	}, {
		name:       "sum + rate: rest client request 5xx",
		expression: `sum(rate(rest_client_requests_total{job=\"kube-controller-manager\", instance=~\"$instance\",code=~\"5..\"}[5m]))`,
	}, {
		name:       "histogram quantile + sum + rate: client request duration",
		expression: `histogram_quantile(0.99, sum(rate(rest_client_request_duration_seconds_bucket{cluster=\"$cluster\", job=\"kube-controller-manager\", instance=~\"$instance\", verb=\"GET\"}[5m])) by (verb, url, le))`,
	}, {
		name:       "rate: process cpu seconds for kube-controller-manager",
		expression: `rate(process_cpu_seconds_total{cluster=\"$cluster\", job=\"kube-controller-manager\",instance=~\"$instance\"}[5m])`,
	}, {
		name:       "simple gauge: go_goroutines",
		expression: `go_goroutines{cluster=\"$cluster\", job=\"kube-controller-manager\",instance=~\"$instance\"}`,
	}, {
		name:       "call: label values of cluster",
		expression: `label_values(up{job=\"kube-controller-manager\"}, cluster)`,
	}, {
		name:       "call: label values of instance",
		expression: `label_values(up{cluster=\"$cluster\", job=\"kube-controller-manager\"}, instance)`,
	},
}
