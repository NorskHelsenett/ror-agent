package clusteragentclient

import (
	"github.com/NorskHelsenett/ror-agent/common/pkg/controllers/dynamiccontroller"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rorkubernetes"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// defaultDynamicClientHandler is the DynamicClientHandler used by
// StartDynamicClient/WithDynamicClient when no custom handler is supplied. It
// reports added/updated/deleted resources to the client's resource cache.
type defaultDynamicClientHandler struct {
	resourceCache resourcecache.ResourceCacheInterface
}

func newDefaultDynamicClientHandler(resourceCache resourcecache.ResourceCacheInterface) *defaultDynamicClientHandler {
	return &defaultDynamicClientHandler{
		resourceCache: resourceCache,
	}
}

func (h *defaultDynamicClientHandler) GetHandlersForSchema(schema schema.GroupVersionResource) dynamiccontroller.DynamicHandler {
	return &defaultSchemaHandler{
		schema:        schema,
		clientHandler: h,
	}
}

func (h *defaultDynamicClientHandler) sendResource(action rortypes.ResourceAction, input map[string]any) {
	removeUnnecessaryResourceData(input)
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

// removeUnnecessaryResourceData strips fields that should not be persisted to
// ROR from the raw kubernetes object before it is converted to a ROR resource.
// These fields are noisy, potentially large, and are not part of the resource
// model, so they must not bleed into the database.
func removeUnnecessaryResourceData(input map[string]any) {
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

type defaultSchemaHandler struct {
	schema        schema.GroupVersionResource
	clientHandler *defaultDynamicClientHandler
}

func (h *defaultSchemaHandler) GetSchema() schema.GroupVersionResource {
	return h.schema
}

func (h *defaultSchemaHandler) GetHandlers() dynamiccontroller.Resourcehandlers {
	return dynamiccontroller.Resourcehandlers{
		AddFunc:    h.addResource,
		UpdateFunc: h.updateResource,
		DeleteFunc: h.deleteResource,
	}
}

func (h *defaultSchemaHandler) addResource(obj any) {
	if obj == nil {
		return
	}
	h.clientHandler.sendResource(rortypes.K8sActionAdd, obj.(*unstructured.Unstructured).Object)
}

func (h *defaultSchemaHandler) deleteResource(obj any) {
	if obj == nil {
		return
	}
	h.clientHandler.sendResource(rortypes.K8sActionDelete, obj.(*unstructured.Unstructured).Object)
}

func (h *defaultSchemaHandler) updateResource(_ any, obj any) {
	if obj == nil {
		return
	}
	h.clientHandler.sendResource(rortypes.K8sActionUpdate, obj.(*unstructured.Unstructured).Object)
}
