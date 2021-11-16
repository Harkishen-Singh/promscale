// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license

package telemetry

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgtype"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"

	"github.com/timescale/promscale/pkg/log"
	"github.com/timescale/promscale/pkg/pgmodel/model/pgutf8str"
	"github.com/timescale/promscale/pkg/pgxconn"
	"github.com/timescale/promscale/pkg/promql"
	"github.com/timescale/promscale/pkg/util"
)

// ExecPlatform To fill this variable in build time, use linker flags.
// Example: go build -ldflags="-X github.com/timescale/promscale/pkg/telemetry.ExecPlatform=<any_string>" ./cmd/promscale/
var ExecPlatform string

type telemetryEngine struct {
	conn            pgxconn.PgxConn
	promqlEngine    *promql.Engine
	promqlQueryable promql.Queryable
	uuid            [16]byte
	telemetryLock   util.AdvisoryLock
	isHouseKeeper   bool

	metricMux sync.RWMutex
	metrics   []telemetryMetric
}

type Telemetry interface {
	StartRoutineAsync(context.Context)
	// BecomeHousekeeper tries to become a telemetry housekeeper. If it succeeds,
	// the caller must call DoHouseKeepingAsync() otherwise the telemetry advisory
	// lock is unreleased.
	BecomeHousekeeper() (success bool, err error)
	DoHouseKeepingAsync(context.Context) error
	// RegisterMetric registers a Prometheus Gauge or Counter metric for telemetry visitor.
	// It must be called after creating the telemetry engine.
	RegisterMetric(statName string, gaugeOrCounterMetric prometheus.Metric) error
}

const telemetryLockId = 0x2D829A932AAFCEDE // Random.

func NewTelemetryEngine(conn pgxconn.PgxConn, uuid [16]byte, connStr string, promqlEngine *promql.Engine, promqlQueryable promql.Queryable) (Telemetry, error) {
	// Warn the users about telemetry collection.
	log.Warn("msg", "Promscale collects anonymous usage telemetry data for project improvements while being GDPR compliant. "+
		"If you wish not to participate, please turn off sending telemetry in TimescaleDB. "+
		"Details about the process can be found at https://docs.timescale.com/timescaledb/latest/how-to-guides/configuration/telemetry/#disabling-telemetry")
	advisoryLock, err := util.NewPgAdvisoryLock(telemetryLockId, connStr)
	if err != nil {
		return nil, fmt.Errorf("error getting pg-advisory-lock: %w", err)
	}
	engine := &telemetryEngine{
		conn:            conn,
		promqlEngine:    promqlEngine,
		promqlQueryable: promqlQueryable,
		uuid:            uuid,
		telemetryLock:   advisoryLock,
	}
	if err = engine.writeMetadata(); err != nil {
		return nil, fmt.Errorf("writing metadata: %w", err)
	}
	return engine, nil
}

// RegisterMetric registers a counter or gauge metric for telemetry purpose.
func (t *telemetryEngine) RegisterMetric(statsName string, metric prometheus.Metric) error {
	ok, err := isCounterOrGauge(metric)
	if err != nil {
		return fmt.Errorf("is counter or gauge: %w", err)
	}
	if !ok {
		return fmt.Errorf("metric should only be of type Counter or Gauge")
	}
	t.metricMux.Lock()
	defer t.metricMux.Unlock()
	t.metrics = append(t.metrics, telemetryMetric{stat: statsName, metric: metric})
	return nil
}

func isCounterOrGauge(metric prometheus.Metric) (bool, error) {
	var temp io_prometheus_client.Metric
	if err := metric.Write(&temp); err != nil {
		return false, fmt.Errorf("writing metric: %w", err)
	}
	if temp.Gauge != nil {
		return true, nil
	} else if temp.Counter != nil {
		return true, nil
	}
	return false, nil
}

func (t *telemetryEngine) StartRoutineAsync(ctx context.Context) {
	go t.telemetrySync(ctx)
}

type telemetryStats struct {
	samplesIngested                float64
	promqlQueriesExecuted          float64
	promqlQueriesTimedout          float64
	promqlQueriesFailed            float64
	traceQueriesExecuted           float64
	traceDependencyQueriesExecuted float64
}

func (t *telemetryEngine) telemetrySync(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
		}
		t.metricMux.RLock()
		if len(t.metrics) == 0 {
			t.metricMux.RUnlock()
			continue
		}
		newStats := telemetryStats{}
		if err := t.metricsVisitor(&newStats); err != nil {
			log.Debug("msg", "error extracting underlying value of a stats metric", "error", err.Error())
			continue
		}
		if err := syncInfoTable(t.conn, t.uuid, newStats); err != nil {
			log.Debug("msg", "syncing new stats", "error", err.Error())
		}

		if !t.isHouseKeeper {
			// Check if housekeeper has died. If yes, then become the housekeeper.
			success, err := t.BecomeHousekeeper()
			if err != nil {
				log.Info("msg", "cannot take the position of telemetry housekeeper", "error", err.Error())
				continue
			}
			if success {
				// Now we are the housekeeper. Use the context of telemetrySync for graceful shutdowns.
				if err = t.DoHouseKeepingAsync(ctx); err != nil {
					log.Error("msg", "unable to do telemetry housekeeping after taking its position", "err", err.Error())
					continue
				}
				log.Info("msg", "Now, I am the telemetry housekeeper")
			}
		}
	}
}

// metricsVisitor fills the provided newStats with most recent metric values.
func (t *telemetryEngine) metricsVisitor(newStats *telemetryStats) error {
	t.metricMux.RLock()
	defer t.metricMux.RUnlock()
	if len(t.metrics) == 0 {
		return nil
	}
	for _, stat := range t.metrics {
		underlyingValue, err := extractMetricValue(stat.metric)
		if err != nil {
			return fmt.Errorf("extracting metric value of stat '%s' with metric '%s': %w", stat.Name(), stat.metric, err)
		}
		switch stat.Name() {
		case "telemetry_metrics_ingested_samples":
			newStats.samplesIngested = underlyingValue
		case "telemetry_metrics_queries_failed":
			newStats.promqlQueriesFailed = underlyingValue
		case "telemetry_metrics_queries_executed":
			newStats.promqlQueriesExecuted = underlyingValue
		case "telemetry_metrics_queries_timed_out":
			newStats.promqlQueriesTimedout = underlyingValue
		case "telemetry_traces_queries_executed":
			newStats.traceQueriesExecuted = underlyingValue
		case "telemetry_traces_dependency_queries_executed":
			newStats.traceDependencyQueriesExecuted = underlyingValue
		default:
			log.Error("msg", fmt.Sprintf("unregistered stat name: %s", stat.Name()))
		}
	}
	return nil
}

func (t *telemetryEngine) BecomeHousekeeper() (success bool, err error) {
	acquired, err := t.telemetryLock.GetAdvisoryLock()
	if err != nil {
		return false, fmt.Errorf("attemping telemetry pg-advisory-lock: %w", err)
	}
	if acquired {
		t.isHouseKeeper = true
	}
	return acquired, nil
}

// DoHouseKeepingAsync starts telemetry housekeeping activities async. It must be called after calling BecomeHousekeeper
// and only when the returned result is success.
func (t *telemetryEngine) DoHouseKeepingAsync(ctx context.Context) error {
	if !t.isHouseKeeper {
		return fmt.Errorf("cannot do house keeping as not a house-keeper")
	}
	go t.housekeeping(ctx)
	return nil
}

// housekeeping takes all the telemetry stats, evaluates and writes into the database.
func (t *telemetryEngine) housekeeping(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 12):
		}
		newStats := make(Stats)
		if err := t.telemetryVisitor(newStats); err != nil {
			log.Debug("msg", "error visiting telemetry", "err", err.Error())
		}
		if len(newStats) > 0 {
			if err := syncTimescaleMetadataTable(t.conn, newStats); err != nil {
				log.Debug("msg", "syncing new stats", "error", err.Error())
			}
		}
		if err := cleanStalePromscalesAfterCounterReset(t.conn); err != nil {
			log.Error("msg", "unable to clean stale Promscale instances. Please report to Promscale team as an issue at https://github.com/timescale/promscale/issues/new",
				"err", err.Error())
		}
	}
}

func (t *telemetryEngine) telemetryVisitor(newStats Stats) (err error) {
	for _, stat := range telemetries {
		switch n := stat.(type) {
		case telemetrySQL:
			query := stat.Query().(string)
			switch n.typ {
			case isInt:
				tmp := new(int64)
				if err := t.conn.QueryRow(context.Background(), query).Scan(&tmp); err != nil {
					return fmt.Errorf("isInt: scanning sql stats '%s' with query '%s': %w", stat.Name(), query, err)
				}
				if tmp == nil {
					continue
				}
				newStats[stat.Name()] = strconv.Itoa(int(*tmp))
			case isString:
				tmp := new(string)
				if err := t.conn.QueryRow(context.Background(), query).Scan(&tmp); err != nil {
					return fmt.Errorf("isString: scanning sql stats '%s' with query '%s': %w", stat.Name(), query, err)
				}
				if tmp == nil {
					continue
				}
				newStats[stat.Name()] = *tmp
			}
		case telemetryPromQL:
			query := stat.Query().(string)
			parsedQuery, err := t.promqlEngine.NewInstantQuery(t.promqlQueryable, query, time.Now())
			if err != nil {
				return fmt.Errorf("error parsing PromQL stats query for stat '%s' with query '%s': %w", stat.Name(), query, err)
			}
			res := parsedQuery.Exec(context.Background())
			if res.Err != nil {
				return fmt.Errorf("error evaluating PromQL stats query for stat '%s' with query '%s': %w", stat.Name(), query, err)
			} else {
				vec, err := res.Vector()
				if err != nil {
					return fmt.Errorf("error getting vector value of PromQL query for stat '%s' with query '%s': %w", stat.Name(), query, err)
				}
				// Ensure the evaluated vector has only one or no points.
				numSamples := len([]promql.Sample(vec))
				if numSamples > 1 {
					return fmt.Errorf("more than one sample receied for stat '%s': received %d samples", stat.Name(), numSamples)
				}
				// Note: numSamples can be zero, indicating that metric does not exist.
				var value float64
				if numSamples == 1 {
					value = []promql.Sample(vec)[0].V
				}
				newStats[stat.Name()] = fmt.Sprint(value)
			}
		default:
			log.Debug("msg", "invalid telemetry type for housekeeper. Expected telemetrySQL or telemetryPromQL", "received", reflect.TypeOf(stat))
		}
	}
	return nil
}

func cleanStalePromscalesAfterCounterReset(conn pgxconn.PgxConn) error {
	_, err := conn.Exec(context.Background(), "SELECT _ps_catalog.clean_stale_promscales_after_counter_reset()")
	return err
}

// writeMetadata writes Promscale and Tobs metadata. Must be written only by
func (t *telemetryEngine) writeMetadata() error {
	promscale, err := promscaleMetadata()
	if err != nil {
		return fmt.Errorf("promscale metadata: %w", err)
	}
	promscale["promscale_exec_platform"] = ExecPlatform
	if err := syncTimescaleMetadataTable(t.conn, Stats(promscale)); err != nil {
		return fmt.Errorf("writing metadata for promscale: %w", err)
	}

	tobs := tobsMetadata()
	if len(tobs) > 0 {
		if err := syncTimescaleMetadataTable(t.conn, Stats(tobs)); err != nil {
			return fmt.Errorf("writing metadata for tobs: %w", err)
		}
	}
	return nil
}

// syncTimescaleMetadataTable syncs the metadata/stats with telemetry metadata table and returns the last error if any.
func syncTimescaleMetadataTable(conn pgxconn.PgxConn, m Stats) error {
	batch := conn.NewBatch()
	for key, metadata := range m {
		safe := pgutf8str.Text{}
		if err := safe.Set(metadata); err != nil {
			return fmt.Errorf("setting in pgutf8 safe string: %w", err)
		}
		query := "INSERT INTO _timescaledb_catalog.metadata VALUES ( $1, $2, true ) ON CONFLICT (key) DO UPDATE SET value = $2, include_in_telemetry = true WHERE metadata.key = $1"
		batch.Queue(query, key, safe)
	}
	results, err := conn.SendBatch(context.Background(), batch)
	if err != nil {
		return fmt.Errorf("error sending batch: %w", err)
	}
	rows, err := results.Query()
	if err != nil {
		return fmt.Errorf("error querying results: %w", err)
	}
	for rows.Next() {
		if err != nil {
			err = rows.Err()
		}
		rows.Close()
	}
	return err
}

// syncInfoTable stats with promscale_instance_information table.
func syncInfoTable(conn pgxconn.PgxConn, uuid [16]byte, s telemetryStats) error {
	lastUpdated := time.Now()

	pgUUID := new(pgtype.UUID)
	if err := pgUUID.Set(uuid); err != nil {
		return fmt.Errorf("setting pg-uuid: %w", err)
	}

	query := `INSERT INTO _ps_catalog.promscale_instance_information
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFlICT (uuid) DO
		UPDATE SET
			last_updated = $2,
			telemetry_metrics_ingested_samples = $3,
			telemetry_metrics_queries_executed = $4,
			telemetry_metrics_queries_timed_out = $5,
			telemetry_metrics_queries_failed = $6,
			telemetry_traces_queries_executed = $7,
			telemetry_traces_dependency_queries_executed = $8
`

	_, err := conn.Exec(context.Background(), query, pgUUID, lastUpdated,
		s.samplesIngested, s.promqlQueriesExecuted, s.promqlQueriesTimedout, s.promqlQueriesFailed,
		s.traceQueriesExecuted, s.traceDependencyQueriesExecuted)
	if err != nil {
		return fmt.Errorf("executing telemetry sync query: %w", err)
	}
	return nil
}

func extractMetricValue(metric prometheus.Metric) (float64, error) {
	var internal io_prometheus_client.Metric
	if err := metric.Write(&internal); err != nil {
		return 0, fmt.Errorf("error writing metric: %w", err)
	}
	if internal.Gauge != nil {
		return internal.Gauge.GetValue(), nil
	} else if internal.Counter != nil {
		return internal.Counter.GetValue(), nil
	}
	return 0, fmt.Errorf("both Gauge and Counter are nil")
}

type noop struct{}

func NewNoopEngine() Telemetry                              { return noop{} }
func (noop) StartRoutineAsync(context.Context)              {}
func (noop) BecomeHousekeeper() (bool, error)               { return false, nil }
func (noop) DoHouseKeepingAsync(context.Context) error      { return nil }
func (noop) RegisterMetric(string, prometheus.Metric) error { return nil }
