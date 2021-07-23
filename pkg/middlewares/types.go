package middlewares

import (
	"github.com/gorilla/mux"
	"github.com/tigera/es-gateway/pkg/cache"
	"github.com/tigera/es-gateway/pkg/clients/elastic"
)

type Type string

const (
	TypeLog  Type = "log"
	TypeAuth Type = "auth"
	TypeSwap Type = "swap"
)

type HandlerMap map[Type]mux.MiddlewareFunc

func GetHandlerMap(es elastic.Client, cache cache.SecretsCache, realUsername, realPassword string) HandlerMap {
	return HandlerMap{
		TypeLog:  logRequestHandler,
		TypeAuth: NewAuthMiddleware(cache),
		TypeSwap: NewSwapElasticCredMiddlware(cache),
	}
}
