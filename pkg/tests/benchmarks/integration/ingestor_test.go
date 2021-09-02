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

	// We change the start and end time while generating timeseries for each test
	// since this allows newer samples to be ingested, avoiding skipping of duplicates.
	start := int64(1)
	end := int64(100000000)
	increment := int64(100000000)

	withDB(b, benchDatabase, func(db *pgxpool.Pool, t testing.TB) {
		ingestor, err := ingstr.NewPgxIngestorForTests(pgxconn.NewPgxConn(db), nil)
		require.NoError(t, err)
		for i := 0; i < 10; i++ {
			ts := common.GenerateLargeTimeseriesWithStartandEnd(start, end)
			b.Run(fmt.Sprintf("ingestor with empty database: batch %d", i), func(b *testing.B) {
				b.ReportAllocs()
				numInsertables, numMetadata, err := ingestor.Ingest(common.NewWriteRequestWithTs(common.CopyMetrics(ts)))
				require.NoError(t, err)
				require.Equal(t, 0, int(numMetadata))
				require.Equal(t, 300006, int(numInsertables))
				start += increment
				end += increment
			})
		}
	})
}
