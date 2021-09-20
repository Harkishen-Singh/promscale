package jaeger_query

import (
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/timescale/promscale/pkg/pgxconn"
)

const getOperations = `select array_agg(sn.name), array_agg(s.span_kind::text) from
_ps_trace.span_name sn inner join _ps_trace.span s
	on sn.id = s.name_id
where _ps_trace.val_text(s.resource_tags, 'int18')=$1 and s.span_kind=$2` // change int18 => service.name

func operations(ctx context.Context, conn pgxconn.PgxConn, query spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	var operationNames, spanKinds []string
	query.ServiceName = "3"
	query.SpanKind = "SPAN_KIND_CONSUMER"
	if err := conn.QueryRow(ctx, getOperations, query.ServiceName, query.SpanKind).Scan(&operationNames, &spanKinds); err != nil {
		return nil, fmt.Errorf("fetching operations: %w", err)
	}
	if len(operationNames) != len(spanKinds) {
		return nil, fmt.Errorf("entries not same in operation-name and span-kind")
	}
	operations := make([]spanstore.Operation, len(operationNames))
	for i := 0; i < len(operationNames); i++ {
		operations[i].Name = operationNames[i]
		operations[i].SpanKind = spanKinds[i]
	}
	return operations, nil
}
