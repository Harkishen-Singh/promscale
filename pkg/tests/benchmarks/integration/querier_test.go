// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/stretchr/testify/require"
	"github.com/timescale/promscale/pkg/api"
	"github.com/timescale/promscale/pkg/pgclient"
	"github.com/timescale/promscale/pkg/pgmodel/cache"
	"github.com/timescale/promscale/pkg/promql"
	"github.com/timescale/promscale/pkg/tenancy"
)

func BenchmarkQuerier(b *testing.B) {
	startContainer()
	defer terminateContainer()

	conf := &pgclient.Config{
		CacheConfig:             cache.DefaultConfig,
		WriteConnectionsPerProc: 4,
		MaxConnections:          -1,
	}

	queryEngine := getQueryEngine(b)

	withDB(b, benchDatabase, func(db *pgxpool.Pool, t testing.TB) {
		prepareContainer(b)

		// Using a role (prom_modifier or prom_reader) here leads to permission error. Investigate.

		client, err := pgclient.NewClientWithPool(conf, 1, db, tenancy.NewNoopAuthorizer(), false)
		require.NoError(t, err)

		queryable := client.Queryable()

		for _, q := range benchmarkableInstantQueries {
			executable := getQuery(t, q, queryEngine, queryable)
			require.NoError(t, err, q.name)

			var result *promql.Result
			b.Run(q.name, func(b *testing.B) {
				b.ReportAllocs()
				result = executable.Exec(context.Background())
			})
			fmt.Println("result", result.String())
		}
	})
}

func getQuery(t testing.TB, q testQuery, engine *promql.Engine, queryable promql.Queryable) promql.Query {
	var (
		err        error
		executable promql.Query
	)

	step, err := api.ParseDuration("10000")
	require.NoError(t, err)

	if q.start == q.end {
		// Instant query.
		executable, err = engine.NewInstantQuery(queryable, q.expression, timestamp.Time(q.start))
	} else {
		executable, err = engine.NewRangeQuery(queryable, q.expression, timestamp.Time(q.start), timestamp.Time(q.end), step)
	}
	require.NoError(t, err)
	return executable
}
