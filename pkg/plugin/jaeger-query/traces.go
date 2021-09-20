package jaeger_query

import (
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/timescale/promscale/pkg/pgxconn"
	"go.opentelemetry.io/collector/consumer/pdata"
	"time"
)

const (
	getTrace = "select 1"
)

func singleTrace(ctx context.Context, conn pgxconn.PgxConn, traceID model.TraceID) (*model.Trace, error) {
	//traceIDstr := traceID.String()

	trace := new(model.Trace)
	if err := conn.QueryRow(ctx, getTrace, traceID).Scan(trace); err != nil {
		return nil, fmt.Errorf("fetching a trace with %s as ID: %w", traceID.String(), err)
	}
	return trace, nil
}

func prepareTrace() pdata.Traces {
	td := pdata.NewTraces()
	s := td.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	s.SetName("mock_span")
	traceID := pdata.NewTraceID([16]byte{})
	s.SetTraceID(traceID)
	emptySpanID := pdata.NewSpanID([8]byte{})
	s.SetSpanID(emptySpanID)
	startTime := pdata.TimestampFromTime(time.Now())
	s.SetStartTimestamp(startTime)
	endTime := pdata.TimestampFromTime(time.Now().Add(time.Minute))
	s.SetEndTimestamp(endTime)
	s.SetKind(pdata.SpanKindConsumer)

	s.SetParentSpanID(emptySpanID)
	s.SetDroppedAttributesCount(1)
	s.SetDroppedEventsCount(1)
	s.SetDroppedLinksCount(1)
	s.SetTraceState("tracestate_1")
	return td
}

func findTraces(ctx context.Context, conn pgxconn.PgxConn, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	traces := make([]*model.Trace, 0)
	// query
	return traces, nil
}

func findTraceIDs(ctx context.Context, conn pgxconn.PgxConn, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	traceIds := make([]model.TraceID, 0)
	// query
	return traceIds, nil
}
