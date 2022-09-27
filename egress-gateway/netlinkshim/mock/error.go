package mock

import "errors"

var (
	ErrorGeneric          = errors.New("dummy error")
	ErrorNetlinkClosed    = errors.New("attempted operation on closed netlink")
	ErrorNotFound         = errors.New("not found")
	ErrorFileDoesNotExist = errors.New("file does not exist")
	ErrorAlreadyExists    = errors.New("already exists")
	ErrorNotSupported     = errors.New("operation not supported")
)
