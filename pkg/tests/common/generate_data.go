// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license

package common

import (
	"io/ioutil"
	"math/rand"
	"os"

	"github.com/golang/snappy"
	"github.com/timescale/promscale/pkg/pgmodel/model"
	"github.com/timescale/promscale/pkg/prompb"
)

const (
	StartTime    int64 = 1577836800000 // 01/01/2020, 05:30:00
	EndTime      int64 = 1577886800000
	EndTimeLarge int64 = 1587886800000 // 26/04/2020, 13:10:00. Generates 335000/2 samples (GenerateSamples).
)

var wr prompb.WriteRequest

func GenerateSamples(index, timeDelta int64) []prompb.Sample {
	delta := float64(index * 2)
	samples := make([]prompb.Sample, 0, 3)
	i := int64(0)
	time := StartTime + (timeDelta * i)

	for time < EndTimeLarge {
		samples = append(samples, prompb.Sample{
			Timestamp: time,
			Value:     delta * float64(i),
		})
		i++
		time = StartTime + (timeDelta * i)
	}

	return samples
}

func GenerateSmallTimeseries() []prompb.TimeSeries {
	return []prompb.TimeSeries{
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "firstMetric"},
				{Name: "foo", Value: "bar"},
				{Name: "common", Value: "tag"},
				{Name: "empty", Value: ""},
			},
			Samples: []prompb.Sample{
				{Timestamp: 1, Value: 0.1},
				{Timestamp: 2, Value: 0.2},
				{Timestamp: 3, Value: 0.3},
				{Timestamp: 4, Value: 0.4},
				{Timestamp: 5, Value: 0.5},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "secondMetric"},
				{Name: "foo", Value: "baz"},
				{Name: "common", Value: "tag"},
			},
			Samples: []prompb.Sample{
				{Timestamp: 1, Value: 1.1},
				{Timestamp: 2, Value: 1.2},
				{Timestamp: 3, Value: 1.3},
				{Timestamp: 4, Value: 1.4},
				{Timestamp: 5, Value: 1.5},
			},
		},
	}
}

// GenerateLargeTimeseries generates timeseries used for ingesting into
// Prometheus and the connector to verify same results are being returned.
// todo: delete the generateLargeTimeseries()
func GenerateLargeTimeseries() []prompb.TimeSeries {
	metrics := []prompb.TimeSeries{
		{
			Labels: []prompb.Label{
				{Name: "aaa", Value: "000"},
				{Name: model.MetricNameLabelName, Value: "metric_1"},
				{Name: "foo", Value: "bar"},
				{Name: "instance", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_1"},
				{Name: "foo", Value: "bar"},
				{Name: "instance", Value: "2"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_1"},
				{Name: "foo", Value: "bar"},
				{Name: "instance", Value: "3"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_2"},
				{Name: "foo", Value: "bat"},
				{Name: "instance", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_2"},
				{Name: "foo", Value: "bat"},
				{Name: "instance", Value: "2"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_2"},
				{Name: "foo", Value: "bat"},
				{Name: "instance", Value: "3"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_3"},
				{Name: "instance", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "metric_3"},
				{Name: "instance", Value: "2"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "METRIC_4"},
				{Name: "foo", Value: "bar"},
			},
		},
	}

	for i := range metrics {
		metrics[i].Samples = GenerateSamples(int64(i+1), 30000)
	}

	return metrics
}

func GeneratePromLikeLargeTimeseries() []prompb.TimeSeries {
	metrics := []prompb.TimeSeries{
		// Gauges.
		// one_gauge_metric has multiple series, which can be utilized for agg queries.
		// two_gauge_metric & three_gauge_metric has one series, hence it can be used in conjunction with one_gauge_metric
		//		to perform across metric aggs.
		{
			Labels: []prompb.Label{
				{Name: "job", Value: "benchmark"},
				{Name: model.MetricNameLabelName, Value: "one_gauge_metric"},
				{Name: "foo", Value: "bar"},
				{Name: "instance", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: "job", Value: "benchmark"},
				{Name: model.MetricNameLabelName, Value: "one_gauge_metric"},
				{Name: "foo", Value: "bar"},
				{Name: "instance", Value: "11"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "two_gauge_metric"},
				{Name: "foo", Value: "bar"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "2"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "three_gauge_metric"},
				{Name: "foo", Value: "bar"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
			},
		},
		// Counters.
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_counter_total"},
				{Name: "version", Value: "1.15"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "two_counter_total"},
				{Name: "version", Value: "1.17"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "1"},
			},
		},
		// Summaries.
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
				{Name: "quantile", Value: "0"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
				{Name: "quantile", Value: "0.25"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
				{Name: "quantile", Value: "0.5"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
				{Name: "quantile", Value: "0.75"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
				{Name: "quantile", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary_sum"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_summary_count"},
				{Name: "job", Value: "benchmark"},
				{Name: "instance", Value: "3"},
			},
		},
		// Histograms.
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_histogram_bucket"},
				{Name: "instance", Value: "2"},
				{Name: "job", Value: "benchmark"},
				{Name: "le", Value: "0.1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_histogram_bucket"},
				{Name: "instance", Value: "2"},
				{Name: "job", Value: "benchmark"},
				{Name: "le", Value: "1"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_histogram_bucket"},
				{Name: "instance", Value: "2"},
				{Name: "job", Value: "benchmark"},
				{Name: "le", Value: "10"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_histogram_bucket"},
				{Name: "instance", Value: "2"},
				{Name: "job", Value: "benchmark"},
				{Name: "le", Value: "100"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_histogram_sum"},
				{Name: "instance", Value: "2"},
				{Name: "job", Value: "benchmark"},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "one_histogram_count"},
				{Name: "instance", Value: "2"},
				{Name: "job", Value: "benchmark"},
			},
		},
	}

	for i := range metrics {
		metrics[i].Samples = GenerateSamples(int64(i+1), 60000)
	}

	return metrics
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomStr(numChars int) string {
	b := make([]byte, numChars)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func GenerateRandomMetricMetadata(num int) []prompb.MetricMetadata {
	randomMetricType := func() prompb.MetricMetadata_MetricType {
		// Generate any metric type from COUNTER to STATESET.
		return prompb.MetricMetadata_MetricType(rand.Intn(int(prompb.MetricMetadata_STATESET)-int(prompb.MetricMetadata_COUNTER)) + 1)
	}

	data := make([]prompb.MetricMetadata, num)
	prefixMetric := "metric_name_"
	prefixHelp := "help_"
	prefixUnit := "unit_"

	for i := 0; i < num; i++ {
		metadata := prompb.MetricMetadata{
			MetricFamilyName: prefixMetric + randomStr(10),
			Type:             randomMetricType(),
			Unit:             prefixUnit + randomStr(5),
			Help:             prefixHelp + randomStr(50),
		}
		data[i] = metadata
	}
	return data
}

func GenerateSmallMultiTenantTimeseries() ([]prompb.TimeSeries, []string) {
	return []prompb.TimeSeries{
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "firstMetric"},
				{Name: "foo", Value: "bar"},
				{Name: "common", Value: "tag"},
				{Name: "empty", Value: ""},
			},
			Samples: []prompb.Sample{
				{Timestamp: 1, Value: 0.1},
				{Timestamp: 2, Value: 0.2},
				{Timestamp: 3, Value: 0.3},
				{Timestamp: 4, Value: 0.4},
				{Timestamp: 5, Value: 0.5},
			},
		},
		{
			Labels: []prompb.Label{
				{Name: model.MetricNameLabelName, Value: "secondMetric"},
				{Name: "job", Value: "baz"},
				{Name: "ins", Value: "tag"},
			},
			Samples: []prompb.Sample{
				{Timestamp: 1, Value: 2.1},
				{Timestamp: 2, Value: 2.2},
				{Timestamp: 3, Value: 2.3},
				{Timestamp: 4, Value: 2.4},
				{Timestamp: 5, Value: 2.5},
			},
		},
	}, []string{"tenant-a", "tenant-b", "tenant-c"}
}

// generateRealTimeseries is used to read the real-world dataset from an
// external file. The dataset was generated using endpoints that are provided
// by the PromLabs PromQL Compliance Tester:
// https://github.com/promlabs/promql-compliance-tester/
// http://demo.promlabs.com:10000
// http://demo.promlabs.com:10001
// http://demo.promlabs.com:10002
func GenerateRealTimeseries() []prompb.TimeSeries {
	if len(wr.Timeseries) == 0 {
		f, err := os.Open("../testdata/real-dataset.sz")
		if err != nil {
			panic(err)
		}

		compressed, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		data, err := snappy.Decode(nil, compressed)
		if err != nil {
			panic(err)
		}

		err = wr.Unmarshal(data)

		if err != nil {
			panic(err)
		}
	}

	return wr.Timeseries
}
