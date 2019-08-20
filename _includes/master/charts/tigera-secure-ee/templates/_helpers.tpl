{{- define "nodeName" -}}
calico-node
{{- end -}}


{{- define "tigera-secure-ee.manager.tls" -}}
{{- if or .Values.manager.tls.cert .Values.manager.tls.key -}}
{{- $_ := required "Must specify both or neither of ee_manager_cert or ee_manager_key" .Values.manager.tls.cert -}}
{{- $_ := required "Must specify both or neither of ee_manager_cert or ee_manager_key" .Values.manager.tls.key -}}
cert: {{ .Values.manager.tls.cert | b64enc }}
key: {{ .Values.manager.tls.key | b64enc }}
{{- else -}}
{{- /* Make certs generated automatically last 100 years. Why? We're doing this automatically for customers
       who haven't provided their own certificates, meaning they might be blissfully unaware that these certs
       are even in use. If we put it at a "recommended" value like 1 year, there is a reasonable chance that
       a year from when they install, they will not have reissued a new cert, and they will have an outage. That's
       really bad. */ -}}
{{- $ca := genSelfSignedCert "localhost" (list "127.0.0.1") (list) 36500 -}}
cert: {{ $ca.Cert | b64enc }}
key: {{ $ca.Key | b64enc }}
{{- end }}
{{- end }}


{{- define "tigera-secure-ee.elasticsearch.mode" -}}
{{- if or (or (or (or (or (or (or (or (or (or (or (or .Values.elasticsearch.host) .Values.elasticsearch.tls.ca) .Values.elasticsearch.fluentd.password) .Values.elasticsearch.manager.password) .Values.elasticsearch.curator.password) .Values.elasticsearch.compliance.controller.password) .Values.elasticsearch.compliance.reporter.password) .Values.elasticsearch.compliance.snapshotter.password) .Values.elasticsearch.compliance.server.password) .Values.elasticsearch.compliance.benchmarker.password) .Values.elasticsearch.intrusionDetection.password) .Values.elasticsearch.elasticInstaller.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.host -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.tls.ca -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.fluentd.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.manager.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.curator.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.compliance.controller.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.compliance.reporter.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.compliance.snapshotter.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.compliance.server.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.compliance.benchmarker.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.intrusionDetection.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.elasticInstaller.password -}}

{{- if or .Values.elasticsearch.tls.crt .Values.elasticsearch.tls.key -}}
{{- $_ := required "Must specify both or none for proxy auth" .Values.elasticsearch.tls.crt -}}
{{- $_ := required "Must specify both or none for proxy auth" .Values.elasticsearch.tls.key -}}
{{- end -}}

external
{{- else -}}
operator
{{- end -}}
{{- end -}}


{{- define "tigera-secure-ee.resourceLimits" -}}
{{- $component := index . 0 -}}
{{- if or (or (or $component.limits.cpu $component.limits.memory) $component.requests.cpu) $component.requests.memory -}}
resources:
{{- if or $component.limits.cpu $component.limits.memory }}
  limits:
{{- if $component.limits.cpu }}
    cpu: {{ $component.limits.cpu | quote }}
{{- end }}
{{- if $component.limits.memory }}
    memory: {{ $component.limits.memory | quote }}
{{- end }}
{{- end }}
{{- if or $component.requests.cpu $component.requests.memory }}
  requests:
{{- if $component.requests.cpu }}
    cpu: {{ $component.requests.cpu | quote }}
{{- end }}
{{- if $component.requests.memory }}
    memory: {{ $component.requests.memory | quote }}
{{- end }}
{{- end }}
{{ end }}
{{- end -}}

{{- define "tigera-secure-ee.kibanaURL" }}
{{- if .Values.manager.kibanaURL }}
  {{- .Values.manager.kibanaURL }}
{{- else if .Values.kibana.host -}}
  https://{{ .Values.kibana.host }}:{{ .Values.kibana.port }}
{{- else -}}
  http://localhost:5601
{{- end }}
{{- end }}
