{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cnx-voltron.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cnx-voltron.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cnx-voltron.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "cnx-voltron.labels" -}}
k8s-app: {{ include "cnx-voltron.name" . }}
{{- end -}}

{{/*
Generate certificates for voltron
*/}}
{{- define "voltron.gen-certs" -}}
{{- if or .Values.certs.provided.crt .Values.certs.provided.key -}}
{{- $_ := required "Must specify both or neither of voltron_cert or voltron_key" .Values.certs.provided.crt -}}
{{- $_ := required "Must specify both or neither of voltron_cert or voltron_key" .Values.certs.provided.key -}}
voltron.crt: {{ .Values.certs.provided.crt| b64enc }}
voltron.key: {{ .Values.certs.provided.key | b64enc }}
{{- end }}
{{- end -}}
