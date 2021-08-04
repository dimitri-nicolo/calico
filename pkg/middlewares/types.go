// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"github.com/gorilla/mux"
	"github.com/tigera/es-gateway/pkg/cache"
	"github.com/tigera/es-gateway/pkg/metrics"
)

type Type string

const (
	TypeLog  Type = "log"
	TypeAuth Type = "auth"
	TypeSwap Type = "swap"
	TypeMetrics Type = "metrics"
)

type HandlerMap map[Type]mux.MiddlewareFunc

func GetHandlerMap(cache cache.SecretsCache, collector metrics.Collector) HandlerMap {
	return HandlerMap{
		TypeLog:  logRequestHandler,
		TypeAuth: NewAuthMiddleware(cache),
		TypeSwap: NewSwapElasticCredMiddlware(cache),
		TypeMetrics: MetricsCollectionHandler(collector),
	}
}
