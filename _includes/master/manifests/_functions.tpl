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
