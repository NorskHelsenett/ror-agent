package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/v2/internal/services/resourceupdatev2"
	"github.com/NorskHelsenett/ror-agent/v2/pkg/clients/dynamicclient"

	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type handler struct {
}

func NewDynamicClientHandler() dynamicclient.DynamicClientHandler {
	ret := handler{}
	return &ret

}

func (handler) AddResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	resourceupdatev2.SendResource(rortypes.K8sActionAdd, obj.(*unstructured.Unstructured).Object)

	obj = nil

}

func (handler) DeleteResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	resourceupdatev2.SendResource(rortypes.K8sActionDelete, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}

func (handler) UpdateResource(_ any, obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	resourceupdatev2.SendResource(rortypes.K8sActionUpdate, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}
