package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/pkg/controllers/dynamiccontroller"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rorkubernetes"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type dynamicClientHandler struct {
	resourceCache resourcecache.ResourceCacheInterface
}

func NewDynamicClientHandler(resourceCache resourcecache.ResourceCacheInterface) *dynamicClientHandler {
	ret := dynamicClientHandler{
		resourceCache: resourceCache,
	}
	return &ret
}

func (h *dynamicClientHandler) GetHandlersForSchema(schema schema.GroupVersionResource) dynamiccontroller.DynamicHandler {
	schemaHandler := schemaHandler{
		schema:        schema,
		clientHandler: h,
	}
	return &schemaHandler
}

func (h *dynamicClientHandler) sendResource(action rortypes.ResourceAction, input map[string]interface{}) {
	rorres := rorkubernetes.NewResourceFromMapInterface(input)
	err := rorres.SetRorMeta(rortypes.ResourceRorMeta{
		Version:  "v2",
		Ownerref: h.resourceCache.GetOwnerref(),
		Action:   action,
	})
	if err != nil {
		rlog.Error("error setting rormeta", err)
		return
	}

	rorres.GenRorHash()

	if action != rortypes.K8sActionDelete && h.resourceCache.CleanupRunning() {
		h.resourceCache.MarkActive(rorres.GetUID())
	}

	needUpdate := h.resourceCache.CheckUpdateNeeded(rorres.GetUID(), rorres.GetRorHash())
	if needUpdate || action == rortypes.K8sActionDelete {

		h.resourceCache.AddResource(rorres)
	}
}
