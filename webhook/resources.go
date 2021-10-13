package webhook

import (
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCaddyfileConfigMap(namespace string, pod *corev1.Pod, caddyfileContents string) *corev1.ConfigMap {
	// TODO: If pod doesn't have an owner, whatdowedo
	cm := &corev1.ConfigMap{
		// TypeMeta required for server side apply
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       namespace,
			Labels:          pod.GetLabels(),
			OwnerReferences: pod.GetOwnerReferences(),
		},
		Data: map[string]string{
			"Caddyfile": caddyfileContents,
		},
	}

	if len(pod.GetOwnerReferences()) > 0 {
		// Use the name of the owner to avoid creating duplicates
		cm.Name = pod.OwnerReferences[0].Name + "-caddyfile"
		return cm
	}
	if pod.GetName() != "" {
		cm.Name = pod.GetName() + "-caddyfile"
	} else if pod.GenerateName != "" {
		cm.Name = pod.GenerateName + "caddyfile"
	}
	return cm
}

func newCertificate(namespace string, pod *corev1.Pod, dnsNames []string) *certv1.Certificate {
	var name string

	if len(pod.GetOwnerReferences()) > 0 {
		// Use the name of the owner to avoid creating duplicates
		name = pod.OwnerReferences[0].Name + "-tls"
	} else {
		if pod.GetName() != "" {
			name = pod.GetName() + "-tls"
		} else if pod.GenerateName != "" {
			name = pod.GenerateName + "tls"
		}
	}

	return &certv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: certv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          pod.GetLabels(),
			OwnerReferences: pod.GetOwnerReferences(),
		},
		Spec: certv1.CertificateSpec{
			DNSNames:   dnsNames,
			SecretName: name,
			IssuerRef: cmmeta.ObjectReference{
				Name: getIssuerName(pod),
				Kind: getIssuerKind(pod),
			},
			// TODO: Allow overriding more options
			PrivateKey: &certv1.CertificatePrivateKey{
				Algorithm: certv1.ECDSAKeyAlgorithm,
				Size:      256,
			},
		},
	}
}
