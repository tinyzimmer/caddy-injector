package webhook

import (
	"bytes"
	"context"
	"strconv"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,sideEffects=None,admissionReviewVersions=v1,failurePolicy=fail,groups="",resources=pods,verbs=create,versions=v1,name=mutate.caddy.io

type PodCaddyInjector struct {
	client.Client

	decoder *admission.Decoder
}

func (c *PodCaddyInjector) InjectDecoder(d *admission.Decoder) error {
	c.decoder = d
	return nil
}

func (c *PodCaddyInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Decode the pod in the request
	pod := &corev1.Pod{}
	if err := c.decoder.Decode(req, pod); err != nil {
		webhookLog.Error(err, "Could not decode request")
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: err.Error(),
					Reason:  metav1.StatusReasonInvalid,
				},
			},
		}
	}

	// See if this is a pod we should inject into
	if !shouldInject(pod) {
		webhookLog.V(1).Info("Ignoring pod that doesn't have inject annotation", "Name", pod.GetName(), "GenerateName", pod.GenerateName)
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: true,
			},
		}
	}

	// Check first that we can determine the HTTP port
	httpPort, err := getHTTPPort(pod)
	if err != nil {
		msg := "Could not determine HTTP port for pod"
		webhookLog.Error(err, msg)
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: msg,
					Reason:  metav1.StatusReasonInvalid,
				},
			},
		}
	}

	// Next make sure that we can get a caddyfile template
	caddyfileTemplate, err := getCaddyfileTemplate(ctx, c.Client, pod)
	if err != nil {
		webhookLog.Error(err, "Could not get Caddyfile template")
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: "Error loading Caddyfile template: " + err.Error(),
					Reason:  metav1.StatusReasonInvalid,
				},
			},
		}
	}

	// Retrieve the rest of the details
	httpsPort := getHTTPSPort(pod)
	dnsNames := getDNSNames(pod)

	// Check if we are creating cert-manager resources
	externalTLS := useCertManager(pod)
	var certificate *certv1.Certificate
	if externalTLS {
		certificate = newCertificate(req.Namespace, pod, dnsNames)
		if err := c.Patch(ctx, certificate, client.Apply, client.FieldOwner(fieldOwner), client.ForceOwnership); err != nil {
			webhookLog.Error(err, "Error creating TLS certificate")
			return admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Message: "Error creating TLS certificate: " + err.Error(),
						Reason:  metav1.StatusReasonInternalError,
					},
				},
			}
		}
	}

	// Render the Caddyfile
	var caddyfile bytes.Buffer
	if err := caddyfileTemplate.Execute(&caddyfile, map[string]interface{}{
		"Pod":             pod,
		"HTTPPort":        strconv.Itoa(int(httpPort)),
		"HTTPSPort":       strconv.Itoa(int(httpsPort)),
		"DNSNames":        dnsNames,
		"ExternalTLS":     externalTLS,
		"CertificateFile": defaultCertificatePath,
		"KeyFile":         defaultKeyPath,
	}); err != nil {
		webhookLog.Error(err, "Error rendering caddyfile template")
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: "Error rendering caddyfile template, see controller logs for more details",
					Reason:  metav1.StatusReasonInvalid,
				},
			},
		}
	}

	// Create a configmap for the Caddyfile
	cm := newCaddyfileConfigMap(req.Namespace, pod, caddyfile.String())
	if err := c.Patch(ctx, cm, client.Apply, client.FieldOwner(fieldOwner), client.ForceOwnership); err != nil {
		webhookLog.Error(err, "Error applying Caddyfile configmap")
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: "Error creating Caddyfile configmap: " + err.Error(),
					Reason:  metav1.StatusReasonInternalError,
				},
			},
		}
	}

	// Start building out the patches

	patches := []jsonpatch.Operation{
		// ConfigMap Caddyfile Volume
		jsonpatch.NewOperation("add", "/spec/volumes/-", corev1.Volume{
			Name: "caddyfile",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.GetName(),
					},
				},
			},
		}),
	}

	// Check if we need to add a TLS secret
	if externalTLS {
		patches = append(patches, jsonpatch.NewOperation("add", "/spec/volumes/-", corev1.Volume{
			Name: "caddy-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certificate.Spec.SecretName,
				},
			},
		}))
	}

	// Build out the caddy container

	caddyContainer := corev1.Container{
		Name:            "caddy",
		Image:           defaultCaddyImage,
		ImagePullPolicy: "IfNotPresent",
		Command:         []string{"caddy", "run"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "https",
				ContainerPort: httpsPort,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "caddyfile",
				MountPath: defaultCaddyfilePath,
				SubPath:   "Caddyfile",
			},
		},
	}

	if externalTLS {
		caddyContainer.VolumeMounts = append(caddyContainer.VolumeMounts, corev1.VolumeMount{
			Name:      "caddy-tls",
			MountPath: defaultTLSPath,
		})
	}

	patches = append(patches, jsonpatch.NewOperation("add", "/spec/containers/-", caddyContainer))

	// Return the patches
	return admission.Response{
		Patches: patches,
		AdmissionResponse: admissionv1.AdmissionResponse{
			Allowed: true,
		},
	}
}
