package dynamicclienthandler

import (
	"github.com/NorskHelsenett/ror-agent/v2/internal/services/resourceupdatev2"
	"github.com/NorskHelsenett/ror-agent/v2/pkg/clients/dynamicclient"

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
	rawData := obj.(*unstructured.Unstructured)
	resourceupdatev2.SendResource(rortypes.K8sActionAdd, rawData)
}

func (handler) DeleteResource(obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdatev2.SendResource(rortypes.K8sActionDelete, rawData)
}

func (handler) UpdateResource(_ any, obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdatev2.SendResource(rortypes.K8sActionUpdate, rawData)
}
