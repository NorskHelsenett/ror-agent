// THIS FILE IS GENERATED, DO NOT EDIT
// ref: build/generator/main.go
package rorResources

import (
	"fmt"
	apiresourcecontracts "github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
)

// the function determines which model to match the resource to and call prepareResourcePayload to cast the input to the matching model.
func (rj rorResourceJson) getResource(resourceReturn *rorResource) error {
	bytes := []byte(rj)

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Namespace" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceNamespace](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Node" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceNode](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "PersistentVolumeClaim" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourcePersistentVolumeClaim](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "Deployment" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceDeployment](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "storage.k8s.io/v1" && resourceReturn.Kind == "StorageClass" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceStorageClass](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "wgpolicyk8s.io/v1alpha2" && resourceReturn.Kind == "PolicyReport" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourcePolicyReport](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "argoproj.io/v1alpha1" && resourceReturn.Kind == "Application" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceApplication](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "argoproj.io/v1alpha1" && resourceReturn.Kind == "AppProject" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceAppProject](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "cert-manager.io/v1" && resourceReturn.Kind == "Certificate" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceCertificate](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Service" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceService](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Pod" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourcePod](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "ReplicaSet" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceReplicaSet](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "StatefulSet" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceStatefulSet](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "apps/v1" && resourceReturn.Kind == "DaemonSet" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceDaemonSet](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "networking.k8s.io/v1" && resourceReturn.Kind == "Ingress" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceIngress](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "networking.k8s.io/v1" && resourceReturn.Kind == "IngressClass" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceIngressClass](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "VulnerabilityReport" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceVulnerabilityReport](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "ExposedSecretReport" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceExposedSecretReport](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "ConfigAuditReport" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceConfigAuditReport](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "RbacAssessmentReport" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceRbacAssessmentReport](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "aquasecurity.github.io/v1alpha1" && resourceReturn.Kind == "ClusterComplianceReport" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceClusterComplianceReport](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "v1" && resourceReturn.Kind == "Endpoints" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceEndpoints](bytes)
		resourceReturn.Resource = payload
		return err
	}

	if resourceReturn.ApiVersion == "networking.k8s.io/v1" && resourceReturn.Kind == "NetworkPolicy" {
		payload, err := prepareResourcePayload[apiresourcecontracts.ResourceNetworkPolicy](bytes)
		resourceReturn.Resource = payload
		return err
	}

	return fmt.Errorf("no handler found for %s/%s", resourceReturn.ApiVersion, resourceReturn.Kind)
}
