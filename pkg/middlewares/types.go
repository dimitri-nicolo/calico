// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"github.com/gorilla/mux"
	"github.com/tigera/es-gateway/pkg/cache"
)

type Type string

const (
	TypeLog  Type = "log"
	TypeAuth Type = "auth"
	TypeSwap Type = "swap"
)

type HandlerMap map[Type]mux.MiddlewareFunc

func GetHandlerMap(cache cache.SecretsCache) HandlerMap {
	return HandlerMap{
		TypeLog:  logRequestHandler,
		TypeAuth: NewAuthMiddleware(cache),
		TypeSwap: NewSwapElasticCredMiddlware(cache),
	}
}
