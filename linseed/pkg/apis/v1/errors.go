package v1

import (
	"fmt"

	elastic "github.com/olivere/elastic/v7"
)

// BulkError indicates an error performing that occurred as part
// of a bulk create operation.
type BulkError struct {
	Resource string `json:"resource"`
	Type     string `json:"type"`
	Reason   string `json:"reason"`
}

func (e BulkError) Error() string {
	if e.Resource == "" {
		// For some errors, there is no specific resource
		fmtString := "Error during a bulk operation. type=%s reason=%s"
		return fmt.Sprintf(fmtString, e.Type, e.Reason)
	}
	fmtString := "Error creating resource as part of a bulk operation. resource=%s type=%s reason=%s"
	return fmt.Sprintf(fmtString, e.Resource, e.Type, e.Reason)
}

// GetBulkErrors returns a slie of bulk errors from an Elastic BulkResponse,
// if there were any errors.
func GetBulkErrors(resp *elastic.BulkResponse) []BulkError {
	var allErrors []BulkError
	if resp.Errors {
		for _, fail := range resp.Failed() {
			allErrors = append(allErrors, BulkError{
				Resource: fail.Error.ResourceId,
				Type:     fail.Error.Type,
				Reason:   fail.Error.Reason,
			})
		}
	}
	return allErrors
}
