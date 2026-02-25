package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/pkg/controllers/dynamiccontroller"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaHandler struct {
	schema        schema.GroupVersionResource
	clientHandler *dynamicClientHandler
}

func (h *schemaHandler) GetSchema() schema.GroupVersionResource {
	return h.schema
}

func (h *schemaHandler) GetHandlers() dynamiccontroller.Resourcehandlers {
	return dynamiccontroller.Resourcehandlers{
		AddFunc:    h.addResource,
		UpdateFunc: h.updateResource,
		DeleteFunc: h.deleteResource,
	}
}

func (h *schemaHandler) addResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	h.clientHandler.sendResource(rortypes.K8sActionAdd, obj.(*unstructured.Unstructured).Object)

	obj = nil

}

func (h *schemaHandler) deleteResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	h.clientHandler.sendResource(rortypes.K8sActionDelete, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}

func (h *schemaHandler) updateResource(_ any, obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	h.clientHandler.sendResource(rortypes.K8sActionUpdate, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}
