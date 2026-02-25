package dynamicclient

import (
	"fmt"
	"os"

	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rordefs"

	"github.com/NorskHelsenett/ror/pkg/rlog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

type DynamicClientHandler interface {
	AddResource(obj any)
	DeleteResource(obj any)
	UpdateResource(_ any, obj any)
}

func Start(k *kubernetesclient.K8sClientsets, dynamichandler DynamicClientHandler, stop chan struct{}, sigs chan os.Signal) error {
	rlog.Info("Starting dynamic watchers")
	discoveryClient, err := k.GetDiscoveryClient()
	if err != nil {
		rlog.Error("Could not initialize discovery client", err)
		return err
	}
	dynamicClient, err := k.GetDynamicClient()
	if err != nil {
		rlog.Error("Could not initialize dynamic client", err)
		return err
	}

	schemas := getSchemas()
	for _, schema := range schemas {
		check, err := discovery.IsResourceEnabled(discoveryClient, schema)
		if err != nil {
			rlog.Error("Could not query resources from cluster", err)
		}
		if check {
			controller := newDynamicWatcher(dynamichandler, dynamicClient, schema)
			go func() {
				controller.Run(stop)
			}()
		} else {
			errmsg := fmt.Sprintf("Could not register resource %s", schema.Resource)
			rlog.Info(errmsg)
		}
	}
	return nil
}

func getSchemas() []schema.GroupVersionResource {
	return rordefs.Resourcedefs.GetSchemasByType(rordefs.ApiResourceTypeAgent)
}

type DynamicWatcher struct {
	dynInformer cache.SharedIndexInformer
	client      dynamic.Interface
}

func (c *DynamicWatcher) Run(stop <-chan struct{}) {
	// Execute go function
	go c.dynInformer.Run(stop)
}

// Function creates a new dynamic controller to listen for api-changes in provided GroupVersionResource
func newDynamicWatcher(dynamichandler DynamicClientHandler, client dynamic.Interface, resource schema.GroupVersionResource) *DynamicWatcher {
	dynWatcher := &DynamicWatcher{}
	dynInformer := dynamicinformer.NewDynamicSharedInformerFactory(client, 0)
	informer := dynInformer.ForResource(resource).Informer()

	dynWatcher.client = client
	dynWatcher.dynInformer = informer

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    dynamichandler.AddResource,
		UpdateFunc: dynamichandler.UpdateResource,
		DeleteFunc: dynamichandler.DeleteResource,
	})
	if err != nil {
		rlog.Error("Error adding event handler", err)
	}

	return dynWatcher
}
