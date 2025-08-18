package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/v2/internal/clients"
	"github.com/NorskHelsenett/ror-agent/v2/pkg/clients/dynamicclient"

	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rorkubernetes"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type handler struct {
}

func NewDynamicClientHandler() dynamicclient.DynamicClientHandler {
	ret := handler{}
	return &ret

}

func (handler) AddResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	SendResource(rortypes.K8sActionAdd, obj.(*unstructured.Unstructured).Object)

	obj = nil

}

func (handler) DeleteResource(obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	SendResource(rortypes.K8sActionDelete, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}

func (handler) UpdateResource(_ any, obj any) {
	if obj == nil {
		return // Avoid memory leaks if obj is nil
	}
	SendResource(rortypes.K8sActionUpdate, obj.(*unstructured.Unstructured).Object)
	obj = nil // Clear the obj reference to avoid memory leaks

}

func SendResource(action rortypes.ResourceAction, input map[string]interface{}) {
	rorres := rorkubernetes.NewResourceFromMapInterface(input)
	err := rorres.SetRorMeta(rortypes.ResourceRorMeta{
		Version:  "v2",
		Ownerref: clients.RorConfig.CreateOwnerref(),
		Action:   action,
	})
	if err != nil {
		rlog.Error("error setting rormeta", err)
		return
	}

	rorres.GenRorHash()

	if action != rortypes.K8sActionDelete && clients.ResourceCache.CleanupRunning() {
		clients.ResourceCache.MarkActive(rorres.GetUID())
	}

	needUpdate := clients.ResourceCache.CheckUpdateNeeded(rorres.GetUID(), rorres.GetRorHash())
	if needUpdate {

		clients.ResourceCache.AddResource(rorres)
		// if err != nil {
		// 	rlog.Error("error sending resource update to ror, added to retryQeue", err)
		// 	ResourceCache.WorkQeueue.Add(rorres)
		// 	return
		// }

	}

}
