package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/v2/pkg/clients/dynamicclient"

	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rorkubernetes"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type handler struct {
	resourceCache resourcecache.ResourceCacheInterface
}

func NewDynamicClientHandler(resourceCache resourcecache.ResourceCacheInterface) dynamicclient.DynamicClientHandler {
	ret := handler{
		resourceCache: resourceCache,
	}
	return &ret
}

func (h *handler) AddResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	h.SendResource(rortypes.K8sActionAdd, obj.(*unstructured.Unstructured).Object)

	obj = nil

}

func (h *handler) DeleteResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	h.SendResource(rortypes.K8sActionDelete, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}

func (h *handler) UpdateResource(_ any, obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	h.SendResource(rortypes.K8sActionUpdate, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}

func (h *handler) SendResource(action rortypes.ResourceAction, input map[string]interface{}) {
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
	if needUpdate {

		h.resourceCache.AddResource(rorres)
	}

}
