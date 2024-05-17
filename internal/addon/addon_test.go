package addon

import (
	"context"
	"fmt"
	"testing"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	v1 "open-cluster-management.io/api/work/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_AgentHealthProber_Healthy(t *testing.T) {
	fakeKubeClient := fake.NewClientBuilder().Build()

	var replicas int32 = 1
	otelCol := &otelv1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "spoke-otelcol",
			Namespace: "spoke-otelcol",
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Replicas: &replicas,
		},
	}

	err := fakeKubeClient.Create(context.TODO(), otelCol, &client.CreateOptions{})
	require.NoError(t, err)

	readyReplicas := int64(*otelCol.Spec.Replicas)

	healthProber := AgentHealthProber(fakeKubeClient)

	err = healthProber.WorkProber.HealthCheck(v1.ResourceIdentifier{
		Group:     otelCol.APIVersion,
		Resource:  otelCol.Kind,
		Name:      otelCol.Name,
		Namespace: otelCol.Namespace,
	}, v1.StatusFeedbackResult{
		Values: []v1.FeedbackValue{
			{
				Name: "replicas",
				Value: v1.FieldValue{
					Type:    v1.Integer,
					Integer: &readyReplicas,
				},
			},
		},
	})

	require.NoError(t, err)

}

func Test_AgentHealthProber_Unhealthy(t *testing.T) {
	cloDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps",
			Kind:       "deployments",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-logging-operator",
			Namespace: "openshift-logging",
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 0,
		},
	}
	readyReplicas := int64(cloDeployment.Status.ReadyReplicas)

	healthProber := AgentHealthProber(nil)

	err := healthProber.WorkProber.HealthCheck(v1.ResourceIdentifier{
		Group:     cloDeployment.APIVersion,
		Resource:  cloDeployment.Kind,
		Name:      cloDeployment.Name,
		Namespace: cloDeployment.Namespace,
	}, v1.StatusFeedbackResult{
		Values: []v1.FeedbackValue{
			{
				Name: "ReadyReplicas",
				Value: v1.FieldValue{
					Type:    v1.Integer,
					Integer: &readyReplicas,
				},
			},
		},
	})

	klog.Info(err)

	expectedErr := fmt.Errorf("readyReplicas is %d for deployement %s/%s", readyReplicas, cloDeployment.Namespace, cloDeployment.Name)
	require.EqualError(t, err, expectedErr.Error())

}
