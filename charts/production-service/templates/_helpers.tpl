{{- define "production-service.fullname" -}}
production-service
{{- end -}}

{{- define "production-service.labels" -}}
app.kubernetes.io/name: {{ include "production-service.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
