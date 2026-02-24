package dynamichandler

import (
	"github.com/NorskHelsenett/ror-agent/pkg/controllers/dynamiccontroller"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type dynamicClientHandler struct {
}

var Handlers = dynamiccontroller.Resourcehandlers{
	AddFunc:    addResource,
	UpdateFunc: updateResource,
	DeleteFunc: deleteResource,
}

func NewDynamicClientHandler() *dynamicClientHandler {
	return &dynamicClientHandler{}
}

func (d dynamicClientHandler) GetHandlersForSchema(schema schema.GroupVersionResource) dynamiccontroller.DynamicHandler {
	handler := schemaHandler{
		schema:   schema,
		handlers: Handlers,
	}
	return &handler
}
