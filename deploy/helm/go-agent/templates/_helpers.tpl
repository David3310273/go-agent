{{/*
Expand the name of the chart.
*/}}
{{- define "go-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "go-agent.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "go-agent.labels" -}}
helm.sh/chart: {{ include "go-agent.name" . }}-{{ .Chart.Version | replace "+" "_" }}
{{ include "go-agent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: go-agent-system
{{- end }}

{{/*
Selector labels
*/}}
{{- define "go-agent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "go-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Go-Agent Server labels
*/}}
{{- define "go-agent.server.labels" -}}
{{ include "go-agent.labels" . }}
app.kubernetes.io/component: server
{{- end }}