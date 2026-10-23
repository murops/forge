{{- define "forge.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "forge.fullname" -}}
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

{{- define "forge.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "forge.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: forge
{{- end }}

{{/* usage: include "forge.selectorLabels" (list . "api") */}}
{{- define "forge.selectorLabels" -}}
{{- $root := index . 0 -}}
app.kubernetes.io/name: {{ include "forge.name" $root }}
app.kubernetes.io/instance: {{ $root.Release.Name }}
app.kubernetes.io/component: {{ index . 1 }}
{{- end }}

{{- define "forge.componentLabels" -}}
{{ include "forge.labels" (index . 0) }}
app.kubernetes.io/component: {{ index . 1 }}
{{- end }}

{{- define "forge.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "forge.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* usage: include "forge.image" (list . .Values.api.image) */}}
{{- define "forge.image" -}}
{{- $root := index . 0 }}
{{- $image := index . 1 }}
{{- $tag := default $root.Chart.AppVersion $root.Values.image.tag }}
{{- printf "%s/%s:%s" $root.Values.image.registry $image.repository $tag }}
{{- end }}

{{- define "forge.databaseEnv" -}}
- name: DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ required "database.existingSecret is required" .Values.database.existingSecret }}
      key: {{ .Values.database.key }}
- name: LOG_LEVEL
  value: {{ .Values.logLevel | quote }}
{{- end }}

{{/* the backend images run as the distroless nonroot user, set by uid so runAsNonRoot can verify it */}}
{{- define "forge.backendPodSecurityContext" -}}
{{- toYaml (mergeOverwrite (dict "runAsUser" 65532 "runAsGroup" 65532 "fsGroup" 65532) (deepCopy .Values.podSecurityContext)) }}
{{- end }}
