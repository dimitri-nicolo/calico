package elastic

import "github.com/olivere/elastic/v7"

func ErrorType(err error) string {
	if err == nil {
		return ""
	}
	e, ok := err.(*elastic.Error)
	if !ok || e == nil || e.Details == nil {
		return ""
	}
	return e.Details.Type
}

// IsAlreadyExists returns true if the error is a resource_already_exists_exception.
func IsAlreadyExists(err error) bool {
	return ErrorType(err) == "resource_already_exists_exception"
}
