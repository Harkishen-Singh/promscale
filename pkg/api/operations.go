package api

import (
	"context"
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/timescale/promscale/pkg/log"
	"io/ioutil"
	"net/http"
)

func Operations(conf *Config, reader spanstore.Reader) http.Handler {
	hf := corsWrapper(conf, operationsHandler(reader))
	return gziphandler.GzipHandler(hf)
}

func operationsHandler(reader spanstore.Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("operations request")
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("msg", fmt.Errorf("reading body: %w", err))
			respondProtoWithErr(w, http.StatusInternalServerError)
			return
		}
		var request storage_v1.GetOperationsRequest
		if err = request.Unmarshal(b); err != nil {
			log.Error("msg", "unmarshalling request: %w", err)
			respondProtoWithErr(w, http.StatusInternalServerError)
			return
		}
		operations, err := reader.GetOperations(context.Background(), spanstore.OperationQueryParameters{
			ServiceName: request.Service,
			SpanKind:    request.SpanKind,
		})
		if err != nil {
			log.Error("msg", fmt.Errorf("get operations: %w", err))
			respondProtoWithErr(w, http.StatusInternalServerError)
			return
		}
		var response storage_v1.GetOperationsResponse
		response.Operations = operationsToProtoOperations(operations)
		b, err = response.Marshal()
		if err != nil {
			log.Error("msg", fmt.Errorf("marshal operations: %w", err))
			respondProtoWithErr(w, http.StatusInternalServerError)
			return
		}
		fmt.Println("sending operations response as", response)
		respondProto(w, http.StatusOK, b)
	}
}

func operationsToProtoOperations(op []spanstore.Operation) []*storage_v1.Operation {
	s := make([]*storage_v1.Operation, len(op))
	for i := range op {
		sOp := new(storage_v1.Operation)
		sOp.Name = op[i].Name
		sOp.SpanKind = op[i].SpanKind
		s[i] = sOp
	}
	return s
}
