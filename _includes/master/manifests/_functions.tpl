{{- define "nodeName" -}}
{{- if and (eq .Values.network "flannel") (eq .Values.datastore "etcd") -}}
canal-node
{{- else if eq .Values.network "flannel" -}}
canal
{{- else if eq .Values.platform "gke" -}}
calico-node-ee
{{- else -}}
calico-node
{{- end -}}
{{- end -}}


{{- define "variant_name" -}}
{{- if eq .Values.network "flannel" -}}
Canal
{{- else -}}
Calico
{{- end -}}
{{- end -}}


{{- define "typha_service_name" -}}
{{- if eq .Values.platform "gke" -}}
calico-typha-ee
{{- else -}}
calico-typha
{{- end -}}
{{- end -}}


{{- define "calico_node_role_name" -}}
{{- if eq .Values.platform "gke" -}}
calico-node-ee
{{- else -}}
calico-node
{{- end -}}
{{- end -}}


{{- define "calico.etcd.tls" -}}
{{- if or (or .Values.etcd.tls.crt .Values.etcd.tls.ca) .Values.etcd.tls.key -}}
{{- $_ := required "Must specify all or none of etcd_crt, etcd_ca, etcd_key" .Values.etcd.tls.crt -}}
{{- $_ := required "Must specify all or none of etcd_crt, etcd_ca, etcd_key" .Values.etcd.tls.ca -}}
{{- $_ := required "Must specify all or none of etcd_crt, etcd_ca, etcd_key" .Values.etcd.tls.key -}}
true
{{- end -}}
{{- end -}}


{{- define "calico.apiserver.tls" -}}
{{- if or .Values.apiserver.tls.crt .Values.apiserver.tls.key -}}
{{- $_ := required "Must specify both or neither of apiserver crt or apiserver key" .Values.apiserver.tls.crt -}}
{{- $_ := required "Must specify both or neither of apiserver crt or apiserver key" .Values.apiserver.tls.key -}}
true
{{- end -}}
{{- end -}}


{{- define "calico.manager.tls" -}}
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
