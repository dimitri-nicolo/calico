{{- define "nodeName" -}}
{{- if and (eq .Values.network "flannel") (eq .Values.datastore "etcd") -}}
canal-node
{{- else if eq .Values.network "flannel" -}}
canal
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
calico-typha
{{- end -}}


{{- define "calico_node_role_name" -}}
calico-node
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

{{- define "calico.resourceLimits" -}}
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
{{end}}
{{- end -}}
