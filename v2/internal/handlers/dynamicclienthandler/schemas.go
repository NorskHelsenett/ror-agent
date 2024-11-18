package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror/pkg/rorresources/rordefs"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (handler) GetSchemas() []schema.GroupVersionResource {
	return rordefs.GetSchemasByType(rordefs.ApiResourceTypeAgent)
}
