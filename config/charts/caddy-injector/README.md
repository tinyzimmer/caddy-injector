# caddy-injector

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: latest](https://img.shields.io/badge/AppVersion-latest-informational?style=flat-square)

A webhook for injecting caddy sidecars into pods.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity for the deployment. |
| domainSuffix | string | `"cluster.local"` | The cluster domain suffix. Used for determining DNS names of the webhook certificate. |
| fullnameOverride | string | `""` | Override the full name of resources. |
| image.pullPolicy | string | `"IfNotPresent"` | The pull policy for retrieving the image. |
| image.repository | string | `"ghcr.io/tinyzimmer/caddy-injector"` | The repository to pull the image. |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion. |
| imagePullSecrets | list | `[]` | Pull secrets required for accessing the image repository. |
| nameOverride | string | `""` | Override the short name of resources. |
| nodeSelector | object | `{}` | Node selector for the deployment. |
| podAnnotations | object | `{}` | Additional annotations to place on the webhook pods. |
| replicaCount | int | `1` | The number of webhook replicas to run. |
| resources | object | `{}` | Resource requests and limits for the deployment. |
| service.port | int | `443` | The port the cluster service should listen on. |
| service.type | string | `"ClusterIP"` | The type of cluster service to create. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account. |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created. |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template. |
| tolerations | list | `[]` | Tolerations for the deployment. |
| webhook.ignoreNamespaces | list | `["kube-system"]` | Additional namespaces to ignore in the webhook configuration. |
| webhook.ignoreReleaseNamespace | bool | `true` | Ignore the namespace of the `caddy-injector` in the webhook configuration. You only want to disable this if you have consistent replicas running. |
| webhook.matchExpressionsOverride | list | `[]` | Fully override the namespace match expressions used in the webhook configuration. Takes precendence over `ignoreReleaseNamespace` and `ignoreNamespaces`. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.5.0](https://github.com/norwoodj/helm-docs/releases/v1.5.0)
