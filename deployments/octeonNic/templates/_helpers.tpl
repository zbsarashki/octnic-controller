{{/*
Full image name with tag
*/}}
{{- define "octnicupdater.fullimage" -}}
{{- .Values.octnicupdater.repository -}}/{{- .Values.octnicupdater.image -}}:{{- .Values.octnicupdater.version | default .Chart.AppVersion -}}
{{- end }}
