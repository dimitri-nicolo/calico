package middlewares

import "net/http"

type HandlerType string

const (
	HandlerTypeLog  HandlerType = "log"
	HandlerTypeAuth HandlerType = "auth"
)

type Handler func(h http.Handler) http.Handler

type HandlerMap map[HandlerType]Handler
