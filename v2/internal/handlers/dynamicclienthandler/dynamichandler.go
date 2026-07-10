package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/common/pkg/controllers/dynamiccontroller"
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
	removeUnnecessaryData(input)
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

// removeUnnecessaryData strips fields that should not be persisted to ROR from
// the raw kubernetes object before it is converted to a ROR resource. These
// fields are noisy, potentially large, and are not part of the resource model,
// so they must not bleed into the database. Mirrors the v1 agent behavior.
func removeUnnecessaryData(input map[string]any) {
	md, ok := input["metadata"].(map[string]any)
	if !ok || md == nil {
		return
	}

	delete(md, "managedFields")

	if ann, ok := md["annotations"].(map[string]any); ok && ann != nil {
		delete(ann, "kubectl.kubernetes.io/last-applied-configuration")
		if len(ann) == 0 {
			delete(md, "annotations")
		}
	}
}
