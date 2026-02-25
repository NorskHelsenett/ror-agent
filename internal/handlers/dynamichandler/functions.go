package dynamichandler

import (
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func addResource(obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdate.SendResource(apiresourcecontracts.K8sActionAdd, rawData)
}

func deleteResource(obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdate.SendResource(apiresourcecontracts.K8sActionDelete, rawData)
}

func updateResource(_ any, obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdate.SendResource(apiresourcecontracts.K8sActionUpdate, rawData)
}
