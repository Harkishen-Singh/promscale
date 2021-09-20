package jaeger_query

import (
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	jaegertranslator "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
	"github.com/timescale/promscale/pkg/pgxconn"
	"go.opentelemetry.io/collector/model/pdata"
	"time"
)

const (
	getTrace = "select 1"
)

func singleTrace(ctx context.Context, conn pgxconn.PgxConn, traceID storage_v1.GetTraceRequest) (*model.Trace, error) {
	//traceIDstr := traceID.String()

	trace := new(model.Trace)
	//if err := conn.QueryRow(ctx, getTrace, traceID).Scan(trace); err != nil {
	//	return nil, fmt.Errorf("fetching a trace with %s as ID: %w", traceID.String(), err)
	//}
	sample := prepareDemoTrace()
	jaegerTrace, err := toJaeger(sample)
	if err != nil {
		return nil, fmt.Errorf("converting to jaeger trace: %w", err)
	}
	if err = batchToSingleTrace(trace, jaegerTrace); err != nil {
		return nil, fmt.Errorf("batch to single trace: %w", err)
	}
	return trace, nil
}

func batchToSingleTrace(trace *model.Trace, batch []*model.Batch) error {
	if len(batch) == 0 {
		return fmt.Errorf("empty batch")
	}
	if len(batch) > 1 {
		// We are asked to send one trace, since a single TraceID can have only a single element in batch.
		// If more than one, there are semantic issues with this trace, hence error out.
		return fmt.Errorf("a single TraceID must contain a single batch of spans. But, found %d", len(batch))
	}
	trace.Spans = batch[0].Spans
	return nil
}

func toJaeger(pTraces pdata.Traces) ([]*model.Batch, error) {
	jaegerTrace, err := jaegertranslator.InternalTracesToJaegerProto(pTraces)
	if err != nil {
		return nil, fmt.Errorf("internal-traces-to-jaeger-proto: %w", err)
	}
	return jaegerTrace, nil
}

func prepareDemoTrace() pdata.Traces {
	td := pdata.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.SetSchemaUrl("http://schema_url")
	s := rs.InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	s.SetName("mock_span")
	traceID := pdata.NewTraceID([16]byte{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	s.SetTraceID(traceID)
	emptySpanID := pdata.NewSpanID([8]byte{0, 0, 1, 0, 0, 0, 1, 0})
	s.SetSpanID(emptySpanID)
	startTime := pdata.NewTimestampFromTime(time.Now())
	s.SetStartTimestamp(startTime)
	endTime := pdata.NewTimestampFromTime(time.Now().Add(time.Minute))
	s.SetEndTimestamp(endTime)
	s.SetKind(pdata.SpanKindConsumer)

	s.SetParentSpanID(emptySpanID)
	s.SetDroppedAttributesCount(1)
	s.SetDroppedEventsCount(1)
	s.SetDroppedLinksCount(1)
	s.SetTraceState("tracestate_1")
	return td
}

func findTraces(ctx context.Context, conn pgxconn.PgxConn, query *storage_v1.TraceQueryParameters) ([]*model.Trace, error) {
	traces := make([]*model.Trace, 0)
	// query
	return traces, nil
}

func findTraceIDs(ctx context.Context, conn pgxconn.PgxConn, query *storage_v1.TraceQueryParameters) ([]model.TraceID, error) {
	traceIds := make([]model.TraceID, 0)
	// query
	return traceIds, nil
}
