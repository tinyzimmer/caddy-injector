package webhook

import (
	"text/template"

	"github.com/Masterminds/sprig"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Pod annotations used for configuring the webhook action
const (

	// InjectAnnotation instructs to inject a caddy sidecar into the spec based on
	// other present annotations.
	InjectAnnotation = "caddy.io/inject"

	// DNSNamesAnnotation defines the DNS name(s) to use for the TLS certificates.
	DNSNamesAnnotation = "caddy.io/dns-names"

	// CertManagerIssuerAnnotation defines an Issuer (in the same namespace as the pod)
	// to use to generate the TLS certificate.
	CertManagerIssuerAnnotation = "caddy.io/issuer"

	// CertManagerClusterIssuerAnnotation defines a ClusterIssuer to use to generate
	// the TLS certificate.
	CertManagerClusterIssuerAnnotation = "caddy.io/cluster-issuer"

	// HTTPPortAnnotation can be used to explicitly set the HTTP port on the pod if it can not be
	// determined automatically.
	HTTPPortAnnotation = "caddy.io/http-port"

	// HTTPSPortAnnotation is used to set the port that caddy should listen on inside the pod.
	// Defaults to `2015`.
	HTTPSPortAnnotation = "caddy.io/https-port"

	// InlineCaddyfileAnnotation defines an inline Caddyfile to use to configure caddy.
	InlineCaddyfileAnnotation = "caddy.io/caddyfile-inline-template"

	// CaddyfileTemplateAnnotation defines a CaddyfileTemplate to use for configuring caddy.
	CaddyfileTemplateAnnotation = "caddy.io/caddyfile-template"
)

var (
	webhookLog = ctrl.Log.WithName("webhook")

	defaultHTTPSPort       int32 = 2015
	defaultTLSPath               = "/srv/certs"
	defaultCaddyfilePath         = "/srv/Caddyfile"
	defaultCertificatePath       = "/srv/certs/tls.crt"
	defaultKeyPath               = "/srv/certs/tls.key"
	defaultCaddyImage            = "caddy:2"

	fieldOwner = "caddy-injector"
)

var defaultCaddyfileTmpl = template.Must(template.New("caddyfile-default").Funcs(sprig.TxtFuncMap()).Parse(`
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
`))
