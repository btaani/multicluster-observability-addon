package addon

import (
	"fmt"

	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	workapiv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func AgentHealthProber() *agent.HealthProber {
	return &agent.HealthProber{
		Type: agent.HealthProberTypeWork,
		WorkProber: &agent.WorkHealthProber{
			ProbeFields: []agent.ProbeField{
				{
					ResourceIdentifier: workapiv1.ResourceIdentifier{
						Group:     "logging.openshift.io",
						Resource:  "clusterlogforwarders",
						Name:      "instance",
						Namespace: ClusterLoggingNS,
					},
					ProbeRules: []workapiv1.FeedbackRule{
						{
							Type: workapiv1.JSONPathsType,
							JsonPaths: []workapiv1.JsonPath{
								{
									Name: "conditions",
									Path: ".status.conditions",
								},
							},
						},
					},
				},
			},
			HealthCheck: func(identifier workapiv1.ResourceIdentifier, result workapiv1.StatusFeedbackResult) error {
				for _, feedbackValue := range result.Values {
					if feedbackValue.Name == "conditions" {
						klog.Info("CLF conditions found")
					} else {
						conditionsErr := fmt.Errorf("addon clf has unexpected conditions value")
						klog.ErrorS(conditionsErr, "Sub may not have installed clf")
						return conditionsErr
					}
				}
				klog.InfoS("health check successful")
				return nil
			},
		},
	}
}
