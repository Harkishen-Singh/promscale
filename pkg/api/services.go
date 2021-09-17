package api

import (
	"context"
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/jaegertracing/jaeger/proto-gen/storage_v1"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/timescale/promscale/pkg/log"
	"net/http"
)

func Services(conf *Config, reader spanstore.Reader) http.Handler {
	hf := corsWrapper(conf, servicesHandler(reader))
	return gziphandler.GzipHandler(hf)
}

func servicesHandler(reader spanstore.Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("into services")
		services, err := reader.GetServices(context.Background())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err, "fetching services")
			return
		}
		response := storage_v1.GetServicesResponse{Services: services}
		bSlice, err := response.Marshal()
		if err != nil {
			log.Error("msg", "jaeger plugin: "+err.Error())
			respondProtoWithErr(w, http.StatusInternalServerError)
			return
		}
		fmt.Println("sending response as", response)
		respondProto(w, http.StatusOK, bSlice)
	}
}
