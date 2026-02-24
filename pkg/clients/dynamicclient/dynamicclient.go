package dynamicclient

import (
	"fmt"
	"os"

	"github.com/NorskHelsenett/ror-agent/internal/handlers/dynamichandler"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/pkg/controllers/dynamiccontroller"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func Start(client clusteragentclient.RorAgentClientInterface, schemas []schema.GroupVersionResource, stopChan chan struct{}, sigs chan os.Signal) error {
	rlog.Info("Starting dynamic watchers")
	dynamicClient, err := client.GetKubernetesClientset().GetDynamicClient()
	if err != nil {
		rlog.Error("failed to get dynamic client", err)
	}
	discoveryClient, err := client.GetKubernetesClientset().GetDiscoveryClient()
	if err != nil {
		rlog.Error("failed to get discovery client", err)
	}

	for _, schema := range schemas {
		check, err := discovery.IsResourceEnabled(discoveryClient, schema)
		if err != nil {
			rlog.Error("Could not query resources from cluster", err)
		}
		if check {
			controller := dynamiccontroller.NewDynamicController(dynamicClient, dynamichandler.GetHandlersForSchema(schema))

			go func() {
				controller.Run(stopChan)
				sig := <-sigs
				fmt.Println(sig)
				stopChan <- struct{}{}
			}()
		} else {
			errmsg := fmt.Sprintf("Could not register resource %s", schema.Resource)
			rlog.Info(errmsg)
		}
	}
	return nil
}
