// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package v3

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/yalp/jsonpath"
	validator "gopkg.in/go-playground/validator.v9"
	k8sv1 "k8s.io/api/core/v1"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

func validateGlobalThreatFeedSpec(structLevel validator.StructLevel) {
	s := structLevel.Current().Interface().(api.GlobalThreatFeed)
	spec := s.Spec
	if spec.Content == api.ThreatFeedContentDomainNameSet && spec.GlobalNetworkSet != nil {
		structLevel.ReportError(
			reflect.ValueOf(spec.Content),
			"Content",
			"",
			reason(string(api.ThreatFeedContentDomainNameSet)+" does not support syncing with a GlobalNetworkSet"),
			"",
		)
	}

	if len(spec.Description) > api.MaxDescriptionLength {
		message := fmt.Sprintf("Invalid maximum length of characters in description = %d as this should be <= %d", len(spec.Description), api.MaxDescriptionLength)
		structLevel.ReportError(
			reflect.ValueOf(spec.Description),
			"Description",
			"",
			reason(message),
			"",
		)
	}

	if spec.Pull != nil && spec.Pull.HTTP != nil {

		// GlobalThreatFeed can reference ConfigMap and Secret. Referenced Kubernetes resource
		// name must follow this convention: "globalthreatfeed-<GlobalThreatFeed Name>-<name>"
		// This is required to prevent a GlobalThreatFeed from referencing and so reading, any
		// Secret/ConfigMap in the tigera-intrusion-detection namespace.
		longPrefix := api.SecretConfigMapNamePrefix + "-" + s.Name + "-"

		for _, header := range spec.Pull.HTTP.Headers {
			if header.ValueFrom != nil && header.ValueFrom.SecretKeyRef != nil &&
				!strings.HasPrefix(header.ValueFrom.SecretKeyRef.Name, longPrefix) {
				structLevel.ReportError(
					reflect.ValueOf(header.ValueFrom.SecretKeyRef),
					"Content",
					"",
					reason("SecretKeyRef.Name must have prefix: "+api.SecretConfigMapNamePrefix+"-<GlobalThreatFeed.Name>-. Current name: "+
						header.ValueFrom.SecretKeyRef.Name+" .Please change name to "+longPrefix+header.ValueFrom.SecretKeyRef.Name+
						" .Please refer to GlobalThreatFeed documentation section for more information on Secret name format."),
					"",
				)
			}
			if header.ValueFrom != nil && header.ValueFrom.ConfigMapKeyRef != nil &&
				!strings.HasPrefix(header.ValueFrom.ConfigMapKeyRef.Name, longPrefix) {
				structLevel.ReportError(
					reflect.ValueOf(header.ValueFrom.ConfigMapKeyRef),
					"Content",
					"",
					reason("ConfigMapKeyRef.Name must have prefix:: "+api.SecretConfigMapNamePrefix+"-<GlobalThreatFeed.Name>. Current name:"+
						header.ValueFrom.ConfigMapKeyRef.Name+" .Please change name to "+longPrefix+header.ValueFrom.ConfigMapKeyRef.Name+
						" .Please refer to GlobalThreatFeed documentation section for more information on ConfigMap name format."),
					"",
				)
			}
		}
	}
}

func validateFeedFormat(structLevel validator.StructLevel) {
	f := structLevel.Current().Interface().(api.ThreatFeedFormat)

	n := 0
	if f.NewlineDelimited != nil {
		n++
	}
	if f.JSON != nil {
		n++
	}
	if f.CSV != nil {
		n++
	}
	if n > 1 {
		structLevel.ReportError(
			reflect.ValueOf(f),
			"",
			"",
			reason("Multiple formats are not supported"),
			"",
		)
	}

}

func validateFeedFormatJSON(structLevel validator.StructLevel) {
	j := structLevel.Current().Interface().(api.ThreatFeedFormatJSON)

	_, err := jsonpath.Prepare(j.Path)
	if err != nil {
		structLevel.ReportError(
			reflect.ValueOf(j.Path),
			"Path",
			"",
			reason(err.Error()),
			"",
		)
	}
}

func validateFeedFormatCSV(structLevel validator.StructLevel) {
	c := structLevel.Current().Interface().(api.ThreatFeedFormatCSV)

	if c.FieldNum != nil && c.FieldName != "" {
		structLevel.ReportError(
			reflect.ValueOf(c),
			"",
			"",
			reason("fieldNum or fieldName may be specified but not both"),
			"",
		)
	}

	if c.FieldName != "" && c.Header == false {
		structLevel.ReportError(
			reflect.ValueOf(c),
			"",
			"",
			reason("if fieldName is set, header must be set to true"),
			"",
		)
	}

	var delimiter rune
	if c.ColumnDelimiter != "" {
		r := []rune(c.ColumnDelimiter)
		if len(r) != 1 {
			structLevel.ReportError(
				reflect.ValueOf(c.ColumnDelimiter),
				"ColumnDelimiter",
				"",
				reason("column delimiter must be a single character"),
				"",
			)
		} else {
			delimiter = r[0]
			if !validDelim(delimiter) {
				structLevel.ReportError(
					reflect.ValueOf(c.ColumnDelimiter),
					"ColumnDelimiter",
					"",
					reason("Invalid column delimiter"),
					"",
				)
			}
		}
	}

	if c.CommentDelimiter != "" {
		r := []rune(c.CommentDelimiter)
		if len(r) != 1 {
			structLevel.ReportError(
				reflect.ValueOf(c.CommentDelimiter),
				"CommentDelimiter",
				"",
				reason("comment delimiter must be a single character"),
				"",
			)
		} else {
			comment := r[0]
			if !validDelim(comment) {
				structLevel.ReportError(
					reflect.ValueOf(c.CommentDelimiter),
					"CommentDelimiter",
					"",
					reason("Invalid comment delimiter"),
					"",
				)
			}

			if comment == delimiter || comment == api.DefaultCSVDelimiter && delimiter == 0 {
				structLevel.ReportError(
					reflect.ValueOf(c.CommentDelimiter),
					"CommentDelimiter",
					"",
					reason("comment and column delimiters must differ"),
					"",
				)
			}
		}
	}

	if c.DisableRecordSizeValidation && c.RecordSize > 0 {
		structLevel.ReportError(
			reflect.ValueOf(c.CommentDelimiter),
			"RecordSize",
			"",
			reason("disableRecordSizeValidation and recordSize are mutually exclusive"),
			"",
		)
	}
}

func validDelim(r rune) bool {
	return r != '"' && r != '\r' && r != '\n' && utf8.ValidRune(r) && r != utf8.RuneError
}

func validateHTTPHeader(structLevel validator.StructLevel) {
	h := structLevel.Current().Interface().(api.HTTPHeader)
	if h.Value != "" && h.ValueFrom != nil {
		structLevel.ReportError(
			reflect.ValueOf(h.Value),
			"Value",
			"",
			reason("Value cannot be used when ValueFrom is used"),
			"")
	}
}

func validateConfigMapKeyRef(structLevel validator.StructLevel) {
	c := structLevel.Current().Interface().(k8sv1.ConfigMapKeySelector)
	for _, errStr := range k8svalidation.IsQualifiedName(c.Name) {
		structLevel.ReportError(
			reflect.ValueOf(c.Name),
			"ConfigMapKeyRef.Name",
			"",
			reason(errStr),
			"",
		)
	}
	for _, errStr := range k8svalidation.IsConfigMapKey(c.Key) {
		structLevel.ReportError(
			reflect.ValueOf(c.Name),
			"ConfigMapKeyRef.Key",
			"",
			reason(errStr),
			"",
		)
	}
}

func validateSecretKeyRef(structLevel validator.StructLevel) {
	c := structLevel.Current().Interface().(k8sv1.SecretKeySelector)
	for _, errStr := range k8svalidation.IsQualifiedName(c.Name) {
		structLevel.ReportError(
			reflect.ValueOf(c.Name),
			"SecretKeyRef.Name",
			"",
			reason(errStr),
			"",
		)
	}
	for _, errStr := range k8svalidation.IsConfigMapKey(c.Key) {
		structLevel.ReportError(
			reflect.ValueOf(c.Name),
			"SecretKeyRef.Key",
			"",
			reason(errStr),
			"",
		)
	}
}
