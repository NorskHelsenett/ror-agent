// THIS FILE IS GENERATED, DO NOT EDIT
// ref: build/generator/main.go
package rorResources

import (
	"fmt"

	apiresourcecontracts "github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
)

// the function determines which model to match the resource to and call prepareResourcePayloadFromObject to cast the input to the matching model.
func getResourceFromObject(resourceReturn *rorResource, obj map[string]any) error {

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Namespace" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceNamespace](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Node" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceNode](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "PersistentVolumeClaim" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourcePersistentVolumeClaim](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "Deployment" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceDeployment](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "storage.k8s.io/v1" && resourceReturn.Kind == "StorageClass" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceStorageClass](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "wgpolicyk8s.io/v1alpha2" && resourceReturn.Kind == "PolicyReport" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourcePolicyReport](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "argoproj.io/v1alpha1" && resourceReturn.Kind == "Application" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceApplication](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "argoproj.io/v1alpha1" && resourceReturn.Kind == "AppProject" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceAppProject](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "cert-manager.io/v1" && resourceReturn.Kind == "Certificate" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceCertificate](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Service" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceService](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Pod" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourcePod](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "ReplicaSet" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceReplicaSet](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "StatefulSet" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceStatefulSet](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "DaemonSet" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceDaemonSet](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "networking.k8s.io/v1" && resourceReturn.Kind == "Ingress" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceIngress](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "networking.k8s.io/v1" && resourceReturn.Kind == "IngressClass" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceIngressClass](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "VulnerabilityReport" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceVulnerabilityReport](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "ExposedSecretReport" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceExposedSecretReport](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "ConfigAuditReport" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceConfigAuditReport](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "RbacAssessmentReport" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceRbacAssessmentReport](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "ClusterComplianceReport" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceClusterComplianceReport](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Endpoints" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceEndpoints](obj)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "networking.k8s.io/v1" && resourceReturn.Kind == "NetworkPolicy" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceNetworkPolicy](obj)
		resourceReturn.Resource = payload
		return err
	}

	return fmt.Errorf("no handler found for %s/%s", resourceReturn.ApiVersion, resourceReturn.Kind)
}
