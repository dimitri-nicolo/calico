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
{{- if (.Values.osType) and eq .Values.osType "windows" -}}
calico-windows
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

{{- /* Used for docs site; don't encode values of the form <something in brackets> so we can 
       output templates we expect the end user to fill in. */ -}}
{{- define "calico.maybeBase64Encode" -}}
{{- if hasPrefix "<" . -}}
{{ . }}
{{- else -}}
{{ . | b64enc }}
{{- end -}}
{{- end -}}


{{- define "calico.apiserver.tls" -}}
{{- if or (or .Values.apiserver.tls.crt .Values.apiserver.tls.key) .Values.apiserver.tls.cabundle -}}
{{- $_ := required "Must specify all or none of apiserver crt, key, and cabundle" .Values.apiserver.tls.crt -}}
{{- $_ := required "Must specify all or none of apiserver crt, key, and cabundle" .Values.apiserver.tls.key -}}
{{- $_ := required "Must specify all or none of apiserver crt, key, and cabundle" .Values.apiserver.tls.cabundle -}}
true
{{- end -}}
{{- end -}}

{{- define "calico.typha.tls" -}}
{{- if or (or (or (or .Values.typha.tls.typhaCrt .Values.typha.tls.typhaKey) .Values.typha.tls.caBundle) .Values.typha.tls.felixCrt) .Values.typha.tls.felixKey -}}
{{- $_ := required "Must specify all or none of typhaCrt, typhaKey, felixCrt, felixKey and caBundle" .Values.typha.tls.typhaCrt -}}
{{- $_ := required "Must specify all or none of typhaCrt, typhaKey, felixCrt, felixKey and caBundle" .Values.typha.tls.typhaKey -}}
{{- $_ := required "Must specify all or none of typhaCrt, typhaKey, felixCrt, felixKey and caBundle" .Values.typha.tls.felixCrt -}}
{{- $_ := required "Must specify all or none of typhaCrt, typhaKey, felixCrt, felixKey and caBundle" .Values.typha.tls.felixKey -}}
{{- $_ := required "Must specify all or none of typhaCrt, typhaKey, felixCrt, felixKey and caBundle" .Values.typha.tls.caBundle -}}
true
{{- end -}}
{{- end -}}

{{- define "calico.hash" -}}
{{- $name := index . 0 -}}
{{- $context := index . 1 -}}
{{- $last := base $context.Template.Name }}
{{- $wtf := $context.Template.Name | replace $last $name -}}
{{- include $wtf $context | sha256sum | quote -}}
{{- end -}}

{{- define "calico.sec_app_profile" -}}
{{- $component := index . 0 -}}
{{- $containerName := index . 1 -}}
{{- if $component.seccompProfile }}
container.seccomp.security.alpha.kubernetes.io/{{ $containerName }}: {{ $component.seccompProfile }}
{{- end }}
{{- if $component.appArmorProfile }}
container.apparmor.security.beta.kubernetes.io/{{ $containerName }}: {{ $component.appArmorProfile }}
{{- end }}
{{- end -}}
