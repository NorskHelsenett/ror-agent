package dynamichandler

import (
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"
	"github.com/NorskHelsenett/ror-agent/pkg/controllers/dynamiccontroller"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaHandler struct {
	schema   schema.GroupVersionResource
	handlers dynamiccontroller.Resourcehandlers
}

var Handlers = dynamiccontroller.Resourcehandlers{
	AddFunc:    addResource,
	UpdateFunc: updateResource,
	DeleteFunc: deleteResource,
}

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

func GetHandlersForSchema(schema schema.GroupVersionResource) dynamiccontroller.DynamicHandler {
	handler := schemaHandler{
		schema:   schema,
		handlers: Handlers,
	}
	return handler
}

func (h schemaHandler) GetSchema() schema.GroupVersionResource {
	return h.schema
}

func (h schemaHandler) GetHandlers() dynamiccontroller.Resourcehandlers {
	return h.handlers
}
