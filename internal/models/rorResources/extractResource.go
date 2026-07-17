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

	if resourceReturn.ApiVersion == "storage.k8s.io/v1" && resourceReturn.Kind == "StorageClass" {
		payload, err := prepareResourcePayloadFromObject[apiresourcecontracts.ResourceStorageClass](obj)
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

	return fmt.Errorf("no handler found for %s/%s", resourceReturn.ApiVersion, resourceReturn.Kind)
}
