package integration

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
	ingstr "github.com/timescale/promscale/pkg/pgmodel/ingestor"
	"github.com/timescale/promscale/pkg/pgxconn"
	"github.com/timescale/promscale/pkg/tests/common"
)

// Ingestor benchmarks need to be intensive, since behaviour is largely dependent on the type of data ingested.
// We divide our benchmarking logic for ingestor into following types:
// 1. Benchmark with generated samples on empty database.
// 2. Benchmark with generated samples on database with existing data.
// 3. Benchmark with data involving backfilling (writing in past).
// 4. Benchmark with real dataset on empty database.
// 5. Benchmark with real dataset on existing data.
// 6. Benchmark with dynamic series formation on empty database.
// 7. Benchmark with dynamic series formation on existing data.

func BenchmarkIngestorWithEmptyDatabase(b *testing.B) {
	startContainer()
	defer terminateContainer()

	ts := common.GenerateLargeTimeseries()

	withDB(b, benchDatabase, func(db *pgxpool.Pool, t testing.TB) {
		ingestor, err := ingstr.NewPgxIngestorForTests(pgxconn.NewPgxConn(db), nil)
		if err != nil {
			t.Fatal(err)
		}
		b.Run("ingestor with empty database", func(b *testing.B) {
			b.ReportAllocs()
			numInsertables, numMetadata, err := ingestor.Ingest(common.NewWriteRequestWithTs(common.CopyMetrics(ts)))
			require.NoError(b, err)
			require.Equal(t, 0, int(numMetadata))
			fmt.Println("numInsertables", numInsertables)
		})
	})
}
