package plugin

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	"github.com/timescale/promscale/pkg/api"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

type Plugin struct {
	name       string
	url        string
	httpClient *http.Client

	logger hclog.Logger
}

func New(url string, logger hclog.Logger, timeout time.Duration) *Plugin {
	return &Plugin{
		url:        url,
		httpClient: &http.Client{Timeout: timeout},
		logger:     logger,
	}
}

func (p *Plugin) SpanReader() spanstore.Reader {
	return p
}

func (p *Plugin) DependencyReader() dependencystore.Reader {
	return p
}

func (p *Plugin) SpanWriter() spanstore.Writer {
	panic("Use Promscale + OTEL-collector to ingest traces")
}

func (p *Plugin) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	request := storage_v1.GetTraceRequest{
		TraceID: traceID,
	}
	reqSlice, err := request.Marshal()
	if err != nil {
		return nil, wrapErr(api.JaegerQuerySingleTraceEndpoint, fmt.Errorf("marshalling request: %w", err))
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.url+api.JaegerQuerySingleTraceEndpoint, bytes.NewReader(reqSlice))
	if err != nil {
		return nil, wrapErr(api.JaegerQuerySingleTraceEndpoint, fmt.Errorf("creating request: %w", err))
	}
	applyValidityHeaders(req)

	// todo: reduce the below code's redundancy.
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, wrapErr(api.JaegerQuerySingleTraceEndpoint, fmt.Errorf("fetching response: %w", err))
	}
	if err = validateResponse(resp); err != nil {
		return nil, wrapErr(api.JaegerQuerySingleTraceEndpoint, fmt.Errorf("validate response: %w", err))
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, wrapErr(api.JaegerQuerySingleTraceEndpoint, fmt.Errorf("reading response body: %w", err))
	}

	return nil, nil
}

func (p *Plugin) GetServices(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.url+api.JaegerQueryServicesEndpoint, nil)
	if err != nil {
		return nil, wrapErr(api.JaegerQueryServicesEndpoint, fmt.Errorf("creating request: %w", err))
	}
	applyValidityHeaders(req)

	// todo: reduce the below code's redundancy.
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, wrapErr(api.JaegerQueryServicesEndpoint, fmt.Errorf("fetching response: %w", err))
	}
	if err = validateResponse(resp); err != nil {
		return nil, wrapErr(api.JaegerQueryServicesEndpoint, fmt.Errorf("validate response: %w", err))
	}

	bSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, wrapErr(api.JaegerQueryServicesEndpoint, fmt.Errorf("reading response body: %w", err))
	}

	var response storage_v1.GetServicesResponse
	if err = response.Unmarshal(bSlice); err != nil {
		return nil, wrapErr(api.JaegerQueryServicesEndpoint, fmt.Errorf("unmarshalling response: %w", err))
	}
	return response.GetServices(), nil
}

func (p *Plugin) GetOperations(ctx context.Context, query spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	p.logger.Warn("msg", "into operations")
	request := storage_v1.GetOperationsRequest{
		Service:  query.ServiceName,
		SpanKind: query.SpanKind,
	}
	p.logger.Warn("request", request)
	reqSlice, err := request.Marshal()
	if err != nil {
		return nil, wrapErr(api.JaegerQueryOperationsEndpoint, fmt.Errorf("marshalling request: %w", err))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.url+api.JaegerQueryOperationsEndpoint, bytes.NewReader(reqSlice))
	if err != nil {
		return nil, wrapErr(api.JaegerQueryOperationsEndpoint, fmt.Errorf("creating request: %w", err))
	}
	applyValidityHeaders(req)

	response, err := p.httpClient.Do(req)
	if err != nil {
		return nil, wrapErr(api.JaegerQueryOperationsEndpoint, fmt.Errorf("fetching response: %w", err))
	}

	p.logger.Warn("received operations response", "yes")

	bSlice, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, wrapErr(api.JaegerQueryOperationsEndpoint, fmt.Errorf("reading response body: %w", err))
	}

	var resp storage_v1.GetOperationsResponse
	if err = resp.Unmarshal(bSlice); err != nil {
		return nil, wrapErr(api.JaegerQueryOperationsEndpoint, fmt.Errorf("unmarshalling response: %w", err))
	}
	var operations []spanstore.Operation
	if resp.Operations != nil {
		for _, operation := range resp.Operations {
			operations = append(operations, spanstore.Operation{
				Name:     operation.Name,
				SpanKind: operation.SpanKind,
			})
		}
	}
	p.logger.Warn("received operations", operations, "of", resp.Operations)
	return operations, nil
}

func (p *Plugin) FindTraces(ctx context.Context, traceQueryParameters *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	return nil, nil
}

func (p *Plugin) FindTraceIDs(ctx context.Context, traceQueryParameters *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	return nil, nil
}

func (p *Plugin) GetDependencies(ctx context.Context, endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	p.logger.Warn("msg", "GetDependencies is yet to be implemented")
	return nil, nil
}
