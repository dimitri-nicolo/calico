package middlewares

import (
	"github.com/gorilla/mux"
	"github.com/tigera/es-gateway/pkg/clients/elastic"
	"github.com/tigera/es-gateway/pkg/clients/kubernetes"
)

type Type string

const (
	TypeLog  Type = "log"
	TypeAuth Type = "auth"
	TypeSwap Type = "swap"
)

type HandlerMap map[Type]mux.MiddlewareFunc

func GetHandlerMap(es elastic.Client, k8s kubernetes.Client, realUsername, realPassword string) HandlerMap {
	return HandlerMap{
		TypeLog:  logRequestHandler,
		TypeAuth: NewAuthMiddleware(k8s),
		TypeSwap: NewSwapElasticCredMiddlware(k8s),
	}
}
