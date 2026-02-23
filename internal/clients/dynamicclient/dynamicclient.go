package dynamicclient

import (
	"github.com/NorskHelsenett/ror/pkg/rorresources/rordefs"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func InitSchema() []schema.GroupVersionResource {
	return rordefs.GetSchemasByType(rordefs.ApiResourceTypeAgent)
}
