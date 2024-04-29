// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"github.com/gorilla/mux"

	"github.com/projectcalico/calico/es-gateway/pkg/cache"
)

type Type int

const (
	TypeLog Type = iota
	TypeAuth
	TypeSwap
	TypeContentType
	TypeMultiTenant
)

type HandlerMap map[Type]mux.MiddlewareFunc

func GetHandlerMap(cache cache.SecretsCache) HandlerMap {
	return HandlerMap{
		TypeLog:         logRequestHandler,
		TypeAuth:        NewAuthMiddleware(cache),
		TypeSwap:        NewSwapElasticCredMiddlware(cache),
		TypeContentType: RejectUnacceptableContentTypeHandler,
	}
}

func GetChallengerHandlerMap(tenancy *KibanaTenancy) HandlerMap {
	return HandlerMap{
		TypeLog:         logRequestHandler,
		TypeMultiTenant: tenancy.Enforce(),
		TypeContentType: RejectUnacceptableContentTypeHandler,
	}
}
