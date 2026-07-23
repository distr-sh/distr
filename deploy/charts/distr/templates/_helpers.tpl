{{/*
Expand the name of the chart.
*/}}
{{- define "distr.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified postgresql name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "distr.postgresql.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "postgresql" "chartValues" .Values.postgresql "context" $) -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "distr.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "distr.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "distr.labels" -}}
helm.sh/chart: {{ include "distr.chart" . }}
{{ include "distr.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "distr.selectorLabels" -}}
app.kubernetes.io/name: {{ include "distr.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Caddy selector labels

Caddy pods must carry an app.kubernetes.io/name distinct from the hub's, so they are not
matched by the hub Deployment's selector (which is only distr.selectorLabels and is immutable
on upgrade). Otherwise the hub ReplicaSet would adopt and churn the Caddy pods.
*/}}
{{- define "distr.caddy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "distr.name" . }}-caddy
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Caddy common labels
*/}}
{{- define "distr.caddy.labels" -}}
helm.sh/chart: {{ include "distr.chart" . }}
{{ include "distr.caddy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "distr.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "distr.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the PostgreSQL connection URI
*/}}
{{- define "distr.databaseUri" -}}
{{- printf "postgresql://%s:$(DATABASE_PASSWORD)@%s:%s/%s?sslmode=prefer" .Values.postgresql.auth.username (include "distr.databaseHost" .) (include "distr.databasePort" .) .Values.postgresql.auth.database -}}
{{- end -}}

{{/*
Return the PostgreSQL Hostname
*/}}
{{- define "distr.databaseHost" -}}
{{- if .Values.postgresql.enabled }}
  {{- if eq .Values.postgresql.architecture "replication" }}
    {{- printf "%s-%s" (include "distr.postgresql.fullname" .) "primary" | trunc 63 | trimSuffix "-" -}}
  {{- else -}}
    {{- print (include "distr.postgresql.fullname" .) -}}
  {{- end -}}
{{- else -}}
  {{- print .Values.externalDatabase.host -}}
{{- end -}}
{{- end -}}

{{/*
Return the PostgreSQL Port
*/}}
{{- define "distr.databasePort" -}}
{{- if .Values.postgresql.enabled }}
    {{- print .Values.postgresql.primary.service.ports.postgresql -}}
{{- else -}}
    {{- printf "%d" (.Values.externalDatabase.port | int ) -}}
{{- end -}}
{{- end -}}

{{/*
Return the PostgreSQL Secret Name
*/}}
{{- define "distr.databaseSecretName" -}}
{{- if .Values.postgresql.enabled }}
    {{- if .Values.postgresql.auth.existingSecret -}}
    {{- print .Values.postgresql.auth.existingSecret -}}
    {{- else -}}
    {{- print (include "distr.postgresql.fullname" .) -}}
    {{- end -}}
{{- else if .Values.externalDatabase.existingSecret -}}
    {{- print .Values.externalDatabase.existingSecret -}}
{{- else -}}
    {{- printf "%s-%s" (include "distr.fullname" .) "externaldb" -}}
{{- end -}}
{{- end -}}

{{- define "distr.hubEnv" -}}
{{- if .Values.postgresql.enabled }}
- name: DATABASE_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "distr.databaseSecretName" . }}
      key: password
{{- end}}
- name: DATABASE_URL
  {{- if .Values.postgresql.enabled }}
  value: {{ include "distr.databaseUri" . }}
  {{- else if .Values.externalDatabase.uri }}
  value: {{ .Values.externalDatabase.uri }}
  {{- else }}
  valueFrom:
    secretKeyRef:
      name: {{ include "distr.databaseSecretName" . }}
      key: {{ .Values.externalDatabase.existingSecretUriKey }}
  {{- end }}
{{ with .Values.hub.env }}
{{- toYaml . }}
{{- end }}
{{- end }}
