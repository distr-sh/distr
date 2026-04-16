<h1 align="center">
  <a href="https://distr.sh/" target="_blank">
    <img alt="" src="https://github.com/distr-sh/distr/raw/refs/heads/main/frontend/ui/public/distr-logo.svg" style="height: 5em;">
  </a>
  <br>
  Distr
</h1>

<div align="center">

**Software Distribution Platform**

</div>

![Version: 1.11.0](https://img.shields.io/badge/Version-1.11.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.11.0](https://img.shields.io/badge/AppVersion-1.11.0-informational?style=flat-square)

The easiest way to distribute enterprise software

**Homepage:** <https://distr.sh/docs/>

## Prerequisites

[Helm](https://helm.sh) must be installed to use the charts. Please refer to Helm's [documentation](https://helm.sh/docs) to get started.

## Installation

Distr is available as a Helm chart distributed via ghcr.io.
To install Distr in Kubernetes, simply run:

<!-- x-release-please-start-version -->

```shell
helm upgrade --install --wait --namespace distr --create-namespace \
  distr oci://ghcr.io/distr-sh/charts/distr --version 1.11.0 \
  --set postgresql.enabled=true --set minio.enabled=true
```

<!-- x-release-please-end -->

## Requirements

| Repository                               | Name       | Version |
| ---------------------------------------- | ---------- | ------- |
| https://charts.min.io                    | minio      | 5.x.x   |
| oci://registry-1.docker.io/bitnamicharts | postgresql | 16.x.x  |

## Values

| Key                                        | Type   | Default                                          | Description |
| ------------------------------------------ | ------ | ------------------------------------------------ | ----------- |
| affinity                                   | object | `{}`                                             |             |
| autoscaling.enabled                        | bool   | `false`                                          |             |
| autoscaling.maxReplicas                    | int    | `10`                                             |             |
| autoscaling.minReplicas                    | int    | `2`                                              |             |
| autoscaling.targetCPUUtilizationPercentage | int    | `100`                                            |             |
| cronJobs[0].args[0]                        | string | `"cleanup"`                                      |             |
| cronJobs[0].args[1]                        | string | `"DeploymentLogRecord"`                          |             |
| cronJobs[0].labels."distr.sh/job"          | string | `"deployment-log-record-cleanup"`                |             |
| cronJobs[0].name                           | string | `"deployment-log-record-cleanup"`                |             |
| cronJobs[1].args[0]                        | string | `"cleanup"`                                      |             |
| cronJobs[1].args[1]                        | string | `"DeploymentRevisionStatus"`                     |             |
| cronJobs[1].labels."distr.sh/job"          | string | `"deployment-revision-status-cleanup"`           |             |
| cronJobs[1].name                           | string | `"deployment-revision-status-cleanup"`           |             |
| cronJobs[2].args[0]                        | string | `"cleanup"`                                      |             |
| cronJobs[2].args[1]                        | string | `"DeploymentTargetMetrics"`                      |             |
| cronJobs[2].labels."distr.sh/job"          | string | `"deployment-target-metrics-cleanup"`            |             |
| cronJobs[2].name                           | string | `"deployment-target-metrics-cleanup"`            |             |
| externalDatabase.existingSecret            | string | `""`                                             |             |
| externalDatabase.existingSecretUriKey      | string | `"uri"`                                          |             |
| externalDatabase.uri                       | string | `""`                                             |             |
| fullnameOverride                           | string | `""`                                             |             |
| hub.envFrom                                | list   | `[]`                                             |             |
| hub.env[0].name                            | string | `"DISTR_HOST"`                                   |             |
| hub.env[0].value                           | string | `"http://distr.local"`                           |             |
| hub.env[10].name                           | string | `"JWT_SECRET"`                                   |             |
| hub.env[10].value                          | string | `"WQrGMYx4tZdGwKlt0RTrhMzfQ+j1wr6z7oRWfmGlETk="` |             |
| hub.env[11].name                           | string | `"MAILER_FROM_ADDRESS"`                          |             |
| hub.env[11].value                          | string | `"My Distr <noreply@distr.local>"`               |             |
| hub.env[12].name                           | string | `"MAILER_TYPE"`                                  |             |
| hub.env[12].value                          | string | `"smtp"`                                         |             |
| hub.env[13].name                           | string | `"MAILER_SMTP_HOST"`                             |             |
| hub.env[13].value                          | string | `"smtp.example.local"`                           |             |
| hub.env[14].name                           | string | `"MAILER_SMTP_PORT"`                             |             |
| hub.env[14].value                          | string | `"25"`                                           |             |
| hub.env[15].name                           | string | `"USER_EMAIL_VERIFICATION_REQUIRED"`             |             |
| hub.env[15].value                          | string | `"false"`                                        |             |
| hub.env[16].name                           | string | `"METRICS_ENTRIES_MAX_AGE"`                      |             |
| hub.env[16].value                          | string | `"1h"`                                           |             |
| hub.env[17].name                           | string | `"LOG_RECORD_ENTRIES_MAX_COUNT"`                 |             |
| hub.env[17].value                          | string | `"500"`                                          |             |
| hub.env[1].name                            | string | `"REGISTRY_ENABLED"`                             |             |
| hub.env[1].value                           | string | `"true"`                                         |             |
| hub.env[2].name                            | string | `"REGISTRY_HOST"`                                |             |
| hub.env[2].value                           | string | `"pkg.distr.local"`                              |             |
| hub.env[3].name                            | string | `"REGISTRY_S3_BUCKET"`                           |             |
| hub.env[3].value                           | string | `"distr"`                                        |             |
| hub.env[4].name                            | string | `"REGISTRY_S3_REGION"`                           |             |
| hub.env[4].value                           | string | `"local"`                                        |             |
| hub.env[5].name                            | string | `"REGISTRY_S3_ENDPOINT"`                         |             |
| hub.env[5].value                           | string | `"http://distr-registry-minio:9000"`             |             |
| hub.env[6].name                            | string | `"REGISTRY_S3_ACCESS_KEY_ID"`                    |             |
| hub.env[6].value                           | string | `"distr"`                                        |             |
| hub.env[7].name                            | string | `"REGISTRY_S3_SECRET_ACCESS_KEY"`                |             |
| hub.env[7].value                           | string | `"distr123"`                                     |             |
| hub.env[8].name                            | string | `"REGISTRY_S3_USE_PATH_STYLE"`                   |             |
| hub.env[8].value                           | string | `"true"`                                         |             |
| hub.env[9].name                            | string | `"REGISTRY_S3_ALLOW_REDIRECT"`                   |             |
| hub.env[9].value                           | string | `"false"`                                        |             |
| image.pullPolicy                           | string | `"IfNotPresent"`                                 |             |
| image.repository                           | string | `"ghcr.io/distr-sh/distr"`                       |             |
| image.tag                                  | string | `""`                                             |             |
| imagePullSecrets                           | list   | `[]`                                             |             |
| ingress.annotations                        | object | `{}`                                             |             |
| ingress.className                          | string | `""`                                             |             |
| ingress.enabled                            | bool   | `false`                                          |             |
| ingress.hosts[0].host                      | string | `"distr.local"`                                  |             |
| ingress.hosts[0].paths[0].path             | string | `"/"`                                            |             |
| ingress.hosts[0].paths[0].pathType         | string | `"ImplementationSpecific"`                       |             |
| ingress.hosts[0].paths[0].port.name        | string | `"http"`                                         |             |
| ingress.hosts[1].host                      | string | `"pkg.distr.local"`                              |             |
| ingress.hosts[1].paths[0].path             | string | `"/"`                                            |             |
| ingress.hosts[1].paths[0].pathType         | string | `"ImplementationSpecific"`                       |             |
| ingress.hosts[1].paths[0].port.name        | string | `"artifacts"`                                    |             |
| ingress.tls                                | list   | `[]`                                             |             |
| livenessProbe.httpGet.path                 | string | `"/"`                                            |             |
| livenessProbe.httpGet.port                 | string | `"http"`                                         |             |
| minio.buckets[0].name                      | string | `"distr"`                                        |             |
| minio.buckets[0].purge                     | bool   | `false`                                          |             |
| minio.deploymentUpdate.type                | string | `"Recreate"`                                     |             |
| minio.enabled                              | bool   | `false`                                          |             |
| minio.fullnameOverride                     | string | `"distr-registry-minio"`                         |             |
| minio.mode                                 | string | `"standalone"`                                   |             |
| minio.persistence.size                     | string | `"20Gi"`                                         |             |
| minio.replicas                             | int    | `1`                                              |             |
| minio.rootPassword                         | string | `"distr123"`                                     |             |
| minio.rootUser                             | string | `"distr"`                                        |             |
| nameOverride                               | string | `""`                                             |             |
| nodeSelector                               | object | `{}`                                             |             |
| podAnnotations                             | object | `{}`                                             |             |
| podLabels                                  | object | `{}`                                             |             |
| podSecurityContext                         | object | `{}`                                             |             |
| postgresql.architecture                    | string | `"standalone"`                                   |             |
| postgresql.auth.database                   | string | `"distr"`                                        |             |
| postgresql.auth.existingSecret             | string | `""`                                             |             |
| postgresql.auth.password                   | string | `""`                                             |             |
| postgresql.auth.username                   | string | `"distr"`                                        |             |
| postgresql.enabled                         | bool   | `false`                                          |             |
| postgresql.service.ports.postgresql        | int    | `5432`                                           |             |
| readinessProbe.httpGet.path                | string | `"/"`                                            |             |
| readinessProbe.httpGet.port                | string | `"http"`                                         |             |
| replicaCount                               | int    | `2`                                              |             |
| resources                                  | object | `{}`                                             |             |
| securityContext                            | object | `{}`                                             |             |
| service.artifactsPort                      | int    | `8585`                                           |             |
| service.port                               | int    | `8080`                                           |             |
| service.type                               | string | `"ClusterIP"`                                    |             |
| serviceAccount.annotations                 | object | `{}`                                             |             |
| serviceAccount.automount                   | bool   | `true`                                           |             |
| serviceAccount.create                      | bool   | `true`                                           |             |
| serviceAccount.name                        | string | `""`                                             |             |
| tolerations                                | list   | `[]`                                             |             |
| volumeMounts                               | list   | `[]`                                             |             |
| volumes                                    | list   | `[]`                                             |             |

## Maintainers

| Name      | Email | Url                            |
| --------- | ----- | ------------------------------ |
| Glasskube |       | <https://github.com/glasskube> |

---

Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
