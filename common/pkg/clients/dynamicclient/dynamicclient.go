package dynamicclient

import (
	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/common/pkg/controllers/dynamiccontroller"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DynamicClientHandler resolves the resource event handlers used by the
// dynamic client's watchers for a given schema.
//
// Deprecated: use clusteragentclient.DynamicClientHandler directly.
type DynamicClientHandler interface {
	GetHandlersForSchema(schema schema.GroupVersionResource) dynamiccontroller.DynamicHandler
}

// MustStart starts the dynamic resource watchers on client.
//
// Deprecated: call client.StartDynamicClient directly, or pass
// clusteragentclient.WithDynamicClient to NewRorAgentClient/
// MustInitNewRorAgentClient to start it eagerly during client creation.
func MustStart(client clusteragentclient.RorAgentClientInterface, handler DynamicClientHandler, schemas ...schema.GroupVersionResource) {
	if err := client.StartDynamicClient(handler, schemas...); err != nil {
		rlog.Fatal("failed to start dynamic client", err)
	}
}
