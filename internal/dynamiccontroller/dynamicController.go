package dynamiccontroller

import (
	"context"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

type DynamicController struct {
	dynInformer cache.SharedIndexInformer
	client      dynamic.Interface
	resource    schema.GroupVersionResource
	noCache     bool
}

func (c *DynamicController) Run(stop <-chan struct{}) {
	if c.noCache {
		go c.runNoCacheWatcher(stop)
		return
	}
	go c.dynInformer.Run(stop)
}

// Function creates a new dynamic controller to listen for api-changes in provided GroupVersionResource
func NewDynamicController(client dynamic.Interface, resource schema.GroupVersionResource) *DynamicController {
	dynWatcher := &DynamicController{}

	dynWatcher.client = client
	dynWatcher.resource = resource
	dynWatcher.noCache = dynamicWatchNoCacheEnabled()

	if dynWatcher.noCache {
		rlog.Info("dynamic watcher enabled", rlog.Any("gvr", resource.String()), rlog.Any("noCache", dynWatcher.noCache))
		return dynWatcher
	}

	dynInformer := dynamicinformer.NewDynamicSharedInformerFactory(client, 0)
	informer := dynInformer.ForResource(resource).Informer()
	dynWatcher.dynInformer = informer

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addResource,
		UpdateFunc: updateResource,
		DeleteFunc: deleteResource,
	})

	if err != nil {
		rlog.Error("Error adding event handler", err)
	}
	return dynWatcher
}

func dynamicWatchNoCacheEnabled() bool {
	return rorconfig.GetBool(config.DynamicWatchNoCacheEnv)
}

func forceGCAfterInitialListEnabled() bool {
	return rorconfig.GetBool(config.ForceGCAfterInitialListEnv)
}

func forceGCAfterInitialListFreeOSMemoryEnabled() bool {
	return rorconfig.GetBool(config.ForceGCAfterInitialListFreeOSMemoryEnv)
}

func maybeForceGCAfterInitialList(gvr string) {
	if !forceGCAfterInitialListEnabled() {
		return
	}
	rlog.Info("forcing GC after initial no-cache list", rlog.Any("env", config.ForceGCAfterInitialListEnv), rlog.Any("gvr", gvr))
	runtime.GC()
	if forceGCAfterInitialListFreeOSMemoryEnabled() {
		// Attempts to return as much memory as possible to the OS.
		debug.FreeOSMemory()
	}
}

func (c *DynamicController) runNoCacheWatcher(stop <-chan struct{}) {
	// Paged LIST + WATCH loop without informer store.
	// This keeps memory bounded compared to informers that retain full objects.
	backoff := time.Second
	resourceVersion := ""

	for {
		if shouldStop(stop) {
			return
		}

		if resourceVersion == "" {
			rv, ok := c.noCacheInitialList(stop, &backoff)
			if !ok {
				// list failed; retry outer loop
				continue
			}
			resourceVersion = rv
		}

		rv, forceRelist := c.noCacheWatch(stop, resourceVersion, &backoff)
		if shouldStop(stop) {
			return
		}
		if forceRelist {
			resourceVersion = ""
			continue
		}
		resourceVersion = rv
	}
}

func shouldStop(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

func increaseBackoff(backoff time.Duration) time.Duration {
	if backoff < 30*time.Second {
		backoff *= 2
	}
	return backoff
}

func (c *DynamicController) noCacheInitialList(stop <-chan struct{}, backoff *time.Duration) (string, bool) {
	cont := ""
	resourceVersion := ""

	for {
		if shouldStop(stop) {
			return "", false
		}

		list, err := c.client.Resource(c.resource).List(context.Background(), metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			rlog.Error("dynamic no-cache list failed", err, rlog.Any("gvr", c.resource.String()))
			time.Sleep(*backoff)
			*backoff = increaseBackoff(*backoff)
			return "", false
		}
		*backoff = time.Second

		for i := range list.Items {
			obj := &list.Items[i]
			addResource(obj)
		}

		resourceVersion = list.GetResourceVersion()
		cont = list.GetContinue()
		if cont == "" {
			maybeForceGCAfterInitialList(c.resource.String())
			return resourceVersion, true
		}
	}
}

func (c *DynamicController) noCacheWatch(stop <-chan struct{}, resourceVersion string, backoff *time.Duration) (string, bool) {
	w, err := c.client.Resource(c.resource).Watch(context.Background(), metav1.ListOptions{ResourceVersion: resourceVersion, AllowWatchBookmarks: true})
	if err != nil {
		rlog.Error("dynamic no-cache watch failed", err, rlog.Any("gvr", c.resource.String()))
		time.Sleep(*backoff)
		*backoff = increaseBackoff(*backoff)
		return resourceVersion, false
	}
	*backoff = time.Second

	for {
		select {
		case <-stop:
			w.Stop()
			return resourceVersion, false
		case evt, ok := <-w.ResultChan():
			if !ok {
				w.Stop()
				// restart watch loop
				return resourceVersion, false
			}

			u, ok := evt.Object.(*unstructured.Unstructured)
			if !ok || u == nil {
				// Ignore Status/error objects here; reconnect on Error events below
				if evt.Type == "ERROR" {
					w.Stop()
					return "", true
				}
				continue
			}

			if rv := u.GetResourceVersion(); rv != "" {
				resourceVersion = rv
			}

			switch evt.Type {
			case "ADDED":
				addResource(u)
			case "MODIFIED":
				updateResource(nil, u)
			case "DELETED":
				deleteResource(u)
			case "BOOKMARK":
				// only updates resourceVersion
			case "ERROR":
				w.Stop()
				return "", true
			}
		}
	}
}

func addResource(obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdate.SendResource(apiresourcecontracts.K8sActionAdd, rawData)
}

func deleteResource(obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdate.SendResource(apiresourcecontracts.K8sActionDelete, rawData)
}

func updateResource(_ any, obj any) {
	rawData := obj.(*unstructured.Unstructured)
	resourceupdate.SendResource(apiresourcecontracts.K8sActionUpdate, rawData)
}
