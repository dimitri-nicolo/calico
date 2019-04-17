{{- define "nodeName" -}}
calico-node
{{- end -}}


{{- define "tigera-secure-lma.manager.tls" -}}
{{- if or .Values.manager.tls.cert .Values.manager.tls.key -}}
{{- $_ := required "Must specify both or neither of ee_manager_cert or ee_manager_key" .Values.manager.tls.cert -}}
{{- $_ := required "Must specify both or neither of ee_manager_cert or ee_manager_key" .Values.manager.tls.key -}}
cert: {{ .Values.manager.tls.cert | b64enc }}
key: {{ .Values.manager.tls.key | b64enc }}
{{- else -}}
{{- $ca := genSelfSignedCert "localhost" (list "127.0.0.1") (list) 365 -}}
cert: {{ $ca.Cert | b64enc }}
key: {{ $ca.Key | b64enc }}
{{- end }}
{{- end }}


{{- define "tigera-secure-lma.elasticsearch.validate" -}}
{{- if or (or (or (or (or (or (or (or .Values.elasticsearch.host) .Values.elasticsearch.tls.ca) .Values.elasticsearch.fluentd.password) .Values.elasticsearch.manager.password) .Values.elasticsearch.curator.password) .Values.elasticsearch.compliance.password) .Values.elasticsearch.intrusionDetection.password) .Values.elasticsearch.elasticInstaller.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.host -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.kibana.host -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.tls.ca -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.fluentd.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.manager.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.curator.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.compliance.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.intrusionDetection.password -}}
{{- $_ := required "Must specify all or none for secure ES settings" .Values.elasticsearch.elasticInstaller.password -}}

{{- if or .Values.elasticsearch.tls.crt .Values.elasticsearch.tls.key -}}
{{- $_ := required "Must specify both or none for proxy auth" .Values.elasticsearch.tls.crt -}}
{{- $_ := required "Must specify both or none for proxy auth" .Values.elasticsearch.tls.key -}}
{{- end -}}

{{- end -}}
{{- end -}}
