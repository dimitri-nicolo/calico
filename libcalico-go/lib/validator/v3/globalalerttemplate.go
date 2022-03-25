package v3

import (
	"fmt"
	"reflect"
	"strings"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	validator "gopkg.in/go-playground/validator.v9"
)

func validateGlobalAlertTemplate(structLevel validator.StructLevel) {
	validateGlobalAlertTemplateName(structLevel)
}

// disallows copying of globalAlert ADJobs template name
func validateGlobalAlertTemplateName(structLevel validator.StructLevel) {
	specs := structLevel.Current().Interface().(v3.GlobalAlertTemplate).Spec

	if specs.Type == v3.GlobalAlertTypeAnomalyDetection && ADDetectorsSet()[specs.Detector.Name] {
		templateObjName := structLevel.Current().Interface().(v3.GlobalAlertTemplate).Name
		expectedName := GlobalAlertDetectorTemplateNamePrefix + strings.Replace(specs.Detector.Name, "_", "-", -1)
		if templateObjName != expectedName && ADDetectorsGlobalAlertTemplateNameSet()[expectedName] {
			// an attempt to create multiple GlobalAlertTemplates for the given detector should not be allowed
			structLevel.ReportError(
				reflect.ValueOf(templateObjName),
				"Detector",
				"",
				reason(fmt.Sprintf("a GlobalAlertTemplate for Detector %s already exists", specs.Detector.Name)),
				"",
			)
		}
	}
}
