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
{{- end -}}
{{- end -}}
