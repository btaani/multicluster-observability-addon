package addon

import (
	"context"
	"fmt"

	loggingv1 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	workapiv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClfGroup    = "logging.openshift.io"
	ClfResource = "clusterlogforwarders"
	ClfName     = "instance"
)

func NewRegistrationOption(agentName string) *agent.RegistrationOption {
	return &agent.RegistrationOption{
		CSRConfigurations: agent.KubeClientSignerConfigurations(Name, agentName),
		CSRApproveCheck:   utils.DefaultCSRApprover(agentName),
	}
}

func GetObjectKey(configRef []addonapiv1alpha1.ConfigReference, group, resource string) client.ObjectKey {
	var key client.ObjectKey
	for _, config := range configRef {
		if config.ConfigGroupResource.Group != group {
			continue
		}
		if config.ConfigGroupResource.Resource != resource {
			continue
		}

		key.Name = config.Name
		key.Namespace = config.Namespace
		break
	}
	return key
}

func AgentHealthProber(k8sClient client.Client) *agent.HealthProber {
	key := types.NamespacedName{Name: "instance", Namespace: ClusterLoggingNS}
	clf := &loggingv1.ClusterLogForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance",
			Namespace: ClusterLoggingNS,
		},
	}

	err := k8sClient.Get(context.TODO(), key, clf)
	if err != nil {
		klog.Errorf("failed to get clusterLogForwarder resource")
		return nil
	}

	var clfStatusJsonPaths []workapiv1.JsonPath

	for i, c := range clf.Status.Conditions {
		if c.Type == "Ready" {
			clfStatusJsonPaths = append(clfStatusJsonPaths, workapiv1.JsonPath{
				Name: "type",
				Path: fmt.Sprintf(".status.conditions[%d].type", i),
			})
		}
		println(clfStatusJsonPaths[i].Path)
	}

	return &agent.HealthProber{
		Type: agent.HealthProberTypeDeploymentAvailability,
		WorkProber: &agent.WorkHealthProber{
			ProbeFields: []agent.ProbeField{
				{
					ResourceIdentifier: workapiv1.ResourceIdentifier{
						Group:     "opentelemetry.io",
						Resource:  "opentelemetrycollectors",
						Name:      "spoke-otelcol",
						Namespace: CollectorNS,
					},
					ProbeRules: []workapiv1.FeedbackRule{
						{
							Type: workapiv1.JSONPathsType,
							JsonPaths: []workapiv1.JsonPath{
								{
									Name: "replicas",
									Path: ".spec.replicas",
								},
							},
						},
					},
				},
				{
					ResourceIdentifier: workapiv1.ResourceIdentifier{
						Group:     ClfGroup,
						Resource:  ClfResource,
						Name:      ClfName,
						Namespace: ClusterLoggingNS,
					},
					ProbeRules: []workapiv1.FeedbackRule{
						{
							Type:      workapiv1.JSONPathsType,
							JsonPaths: clfStatusJsonPaths,
						},
					},
				},
			},
			HealthCheck: func(identifier workapiv1.ResourceIdentifier, result workapiv1.StatusFeedbackResult) error {
				if len(result.Values) == 0 {
					return fmt.Errorf("no values are probed for %s/%s", identifier.Namespace, identifier.Name)
				}
				if identifier.Resource == ClfResource {
					for _, value := range result.Values {
						if value.Name != "type" {
							continue
						}

						if *value.Value.String == "Ready" {
							return nil
						}

						return fmt.Errorf("status condition type is %s for %s/%s", *value.Value.String, identifier.Namespace, identifier.Name)
					}
					return fmt.Errorf("status condition type is not probed")
				}
				for _, value := range result.Values {
					if value.Name != "replicas" {
						continue
					}

					if *value.Value.Integer >= 1 {
						return nil
					}

					return fmt.Errorf("replicas is %d for %s/%s", *value.Value.Integer, identifier.Namespace, identifier.Name)
				}
				return fmt.Errorf("replicas is not probed")
			},
		},
	}
}
