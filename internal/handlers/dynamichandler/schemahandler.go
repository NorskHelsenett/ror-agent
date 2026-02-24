package dynamichandler

import (
	"github.com/NorskHelsenett/ror-agent/pkg/controllers/dynamiccontroller"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaHandler struct {
	schema   schema.GroupVersionResource
	handlers dynamiccontroller.Resourcehandlers
}

func (h schemaHandler) GetSchema() schema.GroupVersionResource {
	return h.schema
}

func (h schemaHandler) GetHandlers() dynamiccontroller.Resourcehandlers {
	return h.handlers
}
