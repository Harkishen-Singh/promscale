package end_to_end_tests

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"

	promscaleJaeger "github.com/timescale/promscale/pkg/jaeger"
	jaegerquery "github.com/timescale/promscale/pkg/jaeger/query"
	ingstr "github.com/timescale/promscale/pkg/pgmodel/ingestor"
	"github.com/timescale/promscale/pkg/pgxconn"
)

func BenchmarkTraceQueryWithTags(b *testing.B) {
	sampleTraces := generateTestTrace()

	withDB(b, "bench_e2e_trace_query_with_tags", func(db *pgxpool.Pool, t testing.TB) {
		ingestor, err := ingstr.NewPgxIngestorForTests(pgxconn.NewPgxConn(db), nil)
		require.NoError(t, err)
		defer ingestor.Close()

		// Ingest traces into Promscale.
		err = ingestor.IngestTraces(context.Background(), copyTraces(sampleTraces))
		require.NoError(t, err)

		// Start Promscale's HTTP endpoint for Jaeger query.
		router, _, err := buildRouter(db)
		require.NoError(t, err)
		jaegerQuery := jaegerquery.New(pgxconn.NewPgxConn(db), &jaegerquery.DefaultConfig)
		promscaleJaeger.ExtendQueryAPIs(router, pgxconn.NewPgxConn(db), jaegerQuery)

		// Bind to the server port. This must be outside of the goroutine below
		// to prevent the server bind and client connect from racing.
		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		go func() {
			server := http.Server{Handler: router}
			require.NoError(t, server.Serve(listener))
		}()

		promscaleClient := httpClient{"http://" + listener.Addr().String()}

		for _, tc := range traceQueryCases {
			b.ResetTimer()
			b.ReportAllocs()
			getTraces(t, promscaleClient, tc.service, tc.start, tc.end, tc.tag)
		}
	})
}
