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

{{/*
Return the Distr configuration as YAML, to be read back with fromYaml. Values files written before
the "hub" key was renamed to "distr" still carry it there, so anything under "hub" fills in what
"distr" leaves at its default and never overrides a value set under "distr". Helm merges the chart's
own defaults into .Values before rendering, so "set under distr" is read off the values themselves:
an env that differs from distr.defaultEnv, a non-empty envFrom, an enabled scratch volume.
*/}}
{{- define "distr.config" -}}
{{- $values := .Values.distr | default dict -}}
{{- $defaultEnv := dig "defaultEnv" (list) $values -}}
{{- $config := omit (deepCopy $values) "defaultEnv" -}}
{{- with .Values.hub -}}
{{- if and .env (or (not $config.env) (eq (toYaml $config.env) (toYaml $defaultEnv))) -}}
{{- $config = set $config "env" .env -}}
{{- end -}}
{{- if and .envFrom (not $config.envFrom) -}}
{{- $config = set $config "envFrom" .envFrom -}}
{{- end -}}
{{- if and .scratch (not (dig "scratch" "enabled" false $config)) -}}
{{- $config = set $config "scratch" (mergeOverwrite (dig "scratch" (dict) $config) .scratch) -}}
{{- end -}}
{{- end -}}
{{- toYaml $config -}}
{{- end -}}

{{- define "distr.env" -}}
{{- $distr := include "distr.config" . | fromYaml }}
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
{{ with $distr.env }}
{{- toYaml . }}
{{- end }}
{{- end }}
