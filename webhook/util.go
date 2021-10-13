package webhook

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"text/template"

	configv1 "github.com/tinyzimmer/caddy-injector/api/v1"

	"github.com/Masterminds/sprig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func shouldInject(pod *corev1.Pod) bool {
	if annotations := pod.GetAnnotations(); annotations != nil {
		_, ok := annotations[InjectAnnotation]
		return ok
	}
	return false
}

func getDNSNames(pod *corev1.Pod) []string {
	if annotations := pod.GetAnnotations(); annotations != nil {
		if names, ok := annotations[DNSNamesAnnotation]; ok {
			return strings.Split(names, ",")
		}
	}
	return []string{"localhost"}
}

func getHTTPSPort(pod *corev1.Pod) int32 {
	if annotations := pod.GetAnnotations(); annotations != nil {
		if port, ok := annotations[HTTPSPortAnnotation]; ok {
			p, err := strconv.Atoi(port)
			if err == nil {
				return int32(p)
			}
			webhookLog.Error(err, "Could not coerce port annotation to integer, falling back to default")
		}
	}
	return defaultHTTPSPort
}

func getHTTPPort(pod *corev1.Pod) (int32, error) {
	// Check for an annotation override
	if annotations := pod.GetAnnotations(); annotations != nil {
		if port, ok := annotations[HTTPPortAnnotation]; ok {
			p, err := strconv.Atoi(port)
			return int32(p), err
		}
	}

	// If there is only one container and port, use that
	if len(pod.Spec.Containers) == 1 && len(pod.Spec.Containers[0].Ports) == 1 {
		return pod.Spec.Containers[0].Ports[0].ContainerPort, nil
	}

	// Look for a port named "http"
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if strings.ToLower(port.Name) == "http" {
				return port.ContainerPort, nil
			}
		}
	}

	// Give up
	return 0, errors.New("could not determine HTTP port for pod")
}

func getCaddyfileTemplate(ctx context.Context, c client.Client, pod *corev1.Pod) (*template.Template, error) {
	if annotations := pod.GetAnnotations(); annotations != nil {
		name := pod.GetName()
		if name == "" {
			name = pod.GenerateName
		}
		if inlineTemplate, ok := annotations[InlineCaddyfileAnnotation]; ok {
			return template.New(name).Funcs(sprig.TxtFuncMap()).Parse(inlineTemplate)
		}
		if crTemplate, ok := annotations[CaddyfileTemplateAnnotation]; ok {
			var tmpl configv1.CaddyfileTemplate
			if err := c.Get(ctx, types.NamespacedName{Name: crTemplate, Namespace: metav1.NamespaceAll}, &tmpl); err != nil {
				return nil, err
			}
			return template.New(name).Funcs(sprig.TxtFuncMap()).Parse(tmpl.Data)
		}
	}
	return defaultCaddyfileTmpl, nil
}

func getIssuerKind(pod *corev1.Pod) string {
	annotations := pod.GetAnnotations()
	if _, ok := annotations[CertManagerIssuerAnnotation]; ok {
		return "Issuer"
	}
	// Assume cluster issuer since a check was already performed for one
	// of the annotations existing.
	return "ClusterIssuer"
}

func getIssuerName(pod *corev1.Pod) string {
	annotations := pod.GetAnnotations()
	if issuer, ok := annotations[CertManagerIssuerAnnotation]; ok {
		return issuer
	}
	if clusterIssuer, ok := annotations[CertManagerClusterIssuerAnnotation]; ok {
		return clusterIssuer
	}
	// Shouldn't fire since one of the two annotations exists already
	return ""
}

func useCertManager(pod *corev1.Pod) bool {
	if annotations := pod.GetAnnotations(); annotations != nil {
		if _, ok := annotations[CertManagerIssuerAnnotation]; ok {
			return true
		}
		if _, ok := annotations[CertManagerClusterIssuerAnnotation]; ok {
			return true
		}
	}
	return false
}
