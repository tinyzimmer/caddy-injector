# caddy-injector
A webhook for injecting caddy sidecars into Kubernetes Pods

This project is an early PoC and subject to changes.

## Install

For now there are two installation methods provided. A helm chart and a bundled manifest.

It is required that [cert-manager](https://cert-manager.io/docs/installation/) be installed in the cluster at the moment for generating webhook certificates (at least, more details below).

### Manifest

```bash
kubectl apply -f \
    https://github.com/tinyzimmer/caddy-injector/raw/main/config/bundle.yaml
```

The manifest will create a `caddy-system` namespace containing the injector pod, as well as necessary RBAC and MutatingWebhookConfigurations.
You can pre-download the manifest or make patches to your liking.

### Helm

The chart is not published (yet) so you will need to clone this repository first

```bash
git clone https://github.com/tinyzimmer/caddy-injector
cd caddy-injector/config
```

You can then install the chart via its local path:

```bash
helm install caddy-injector charts/caddy-injector [additional args...]
```

To see documentation for the available values consult the chart's [README](config/charts/caddy-injector).

## Usage

The injector works by intercepting matched requests to create pods and checking for any of the following annotations:

 - `caddy.io/inject`: When present and set to _any_ value (e.g. an empty string), injects a caddy proxy into the pod. For any changes to take place, this annotation **must** exist.

 - `caddy.io/dns-names`: A comma-separated list of DNS names to use in the auto-generated TLS certificate. Defaults to "localhost".

 - `caddy.io/issuer`: Instead of using the built-in TLS generator or an acme configuration in the **Caddyfile**, use the given cert-manager **Issuer** in the Pod's namespace to provision a TLS certificate for caddy.

 - `caddy.io/cluster-issuer`: Same as above, except for a cert-manager **ClusterIssuer**.

 - `caddy.io/http-port`: The injector will try to detect the HTTP port by inspecting the containers in the spec. If it finds a single container port, that is chosen. Otherwise it will search for a port named `http`. If either of these auto-detection methods are unfeasible, you can supply the _numerical_ value of the port here.

 - `caddy.io/https-port`: The port that caddy will listen on inside the pod for HTTPS connections. Defaults to "2015".

 - `caddy.io/caddyfile-inline-template`: Provide an inline caddyfile template for this Pod. More information on templates can be found in the [Templates](#caddyfile-templates) section.

 - `caddy.io/caddyfile-template`: Reference a **CaddyfileTemplate** CR for use with this pod.

 To see examples of various usages check the [samples](config/samples) folder.

 ## Caddyfile Templates

 Templates can be defined inline in pod annotations or via the **CaddyfileTemplate** as shown above.
 The template is a `go-template` with [`sprig`](https://masterminds.github.io/sprig/) functions. If you are familiar with helm templating, you'll be right at home.

 The following variables are present during the evaluation of the template:

  - `Pod`: The **Pod** object itself in its Go representation. You can use the [official documentation](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/) to see the values present. The anchor links next to each field contain the name of the Go field.
  
  - `HTTPPort`: The port that traffic is to be proxied to, as a string.

  - `HTTPSPort`: The port that caddy should listen to HTTPS on. This is the same value that is being injected into the pod spec.

  - `DNSNames`: DNS names for the certificate as parsed from the Pod's annotations.

  - `ExternalTLS`: There would be few instances where this is used in a custom template. When a certificate provided by an external source is present (i.e. cert-manager), this will be set to `true`.

  - `CertificateFile`: When `ExternalTLS` is `true`, this will be populated with the path to the server TLS certificate file.

  - `KeyFile`: When `ExternalTLS` is `true`, this will be populated with the path to the server TLS key file.

The default template, which is nearly identical to that used in the examples, is very simple and looks like this:

```go-template
{
    auto_https disable_redirects
    https_port {{ .HTTPSPort }}
}

{{ .DNSNames | join ", " }} {
    {{- if .ExternalTLS }}
    tls {{ .CertificateFile }} {{ .KeyFile }}
    {{- end }}
    reverse_proxy 127.0.0.1:{{ .HTTPPort }}
}
```

Redirects are disabled by default, because often the pod will not have permissions to listen on 80 **or** the port will already be taken by the application itself.

You can supply your own templates via the `caddy.io/caddyfile-inline-template` or `caddy.io/caddyfile-template` annotations described above. The former for inline, and the latter makes use of the CR.

The **CaddyfileTemplate** CR works identical to that of **Secrets** and **ConfigMaps** except instead of a key-value pairs, it expects a single string value.
A representation of the above template in a CR to be shared amnongst Pods would look like this:

```yaml
apiVersion: config.caddy.io/v1
kind: CaddyfileTemplate
metadata:
  name: example-template
data: |
  {
    auto_https disable_redirects
    https_port {{ .HTTPSPort }}
  }

  {{ .DNSNames | join ", " }} {
    {{- if .ExternalTLS }}
    tls {{ .CertificateFile }} {{ .KeyFile }}
    {{- end }}
    reverse_proxy 127.0.0.1:{{ .HTTPPort }}
  }
```

## Trying out Locally with K3d

The `Makefile` contains helpers for development and testing locally using a [`k3d`](https://k3d.io/v5.0.1/#installation) cluster.

To see all available targets you can run `make help`.
To build the injector, start up a cluster, then install cert-manager and the injector itself, you can simply run:

```bash
# You'll need to have make, docker, and k3d installed already
make docker-build full-cluster
```

When the command finishes you'll have a k3s cluster running on your machine where you can test and play around with the injector.

## TODO

- [ ] Docs could use improvement
- [ ] A webhook for validating **CaddyfileTemplates**
- [ ] Tests, tests, tests