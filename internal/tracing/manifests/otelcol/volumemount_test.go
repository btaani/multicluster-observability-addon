package otelcol

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_ConfigureVolumeMounts(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tracing-otlphttp-auth",
			Namespace: "cluster-1",
			Annotations: map[string]string{
				annotation: "otlphttp",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte("data"),
			"ca.crt":  []byte("data"),
			"tls.key": []byte("data"),
		},
	}

	otelCol := v1beta1.OpenTelemetryCollector{}

	ConfigureVolumeMounts(&otelCol, secret)
	require.NotEmpty(t, otelCol.Spec.VolumeMounts)
}
