package clusterhandler

import (
	"context"
	"time"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/NorskHelsenett/ror/pkg/rorresources"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rortypes"
	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func MustStart(agentclient clusteragentclient.RorAgentClientInterface, resourceCacheInterface resourcecache.ResourceCacheInterface) {
	err := Start(agentclient, resourceCacheInterface)
	if err != nil {
		rlog.Fatal("could not start cluster handler", err)
	}
}

func Start(agentclient clusteragentclient.RorAgentClientInterface, resourceCacheInterface resourcecache.ResourceCacheInterface) error {
	rlog.Info("Starting cluster handler", rlog.String("clusterid", agentclient.GetClusterInterregator().GetClusterId()))

	// Get myself
	existing, err := agentclient.GetRorClient().V2().Resources().Get(context.TODO(), rorresources.ResourceQuery{
		VersionKind: rortypes.ResourceKubernetesClusterGVK,
	},
	)
	if err != nil || len(existing.Resources) > 1 {
		rlog.Error("error fetching existing resources for cluster handler", err)
	}

	// Add cluster resource to workqueue to ensure it exists in the system and to trigger any logic related to it
	interregator := agentclient.GetClusterInterregator()
	var clusterresource *rorresources.Resource
	if len(existing.Resources) == 0 {
		clusterresource = rorresources.NewRorKubernetesClusterResource()
		clusterresource.Metadata.UID = types.UID(uuid.NewString())
		clusterresource.Metadata.CreationTimestamp = v1.Now()
		err = clusterresource.SetRorMeta(rortypes.ResourceRorMeta{
			Version:  "v2",
			Ownerref: resourceCacheInterface.GetOwnerref(),
			Action:   rortypes.K8sActionAdd,
		})
		if err != nil {
			return err
		}
	}

	if len(existing.Resources) == 1 {
		clusterresource = rorresources.NewResourceFromStruct(*existing.Resources[0])
		clusterresource.RorMeta.Action = rortypes.K8sActionUpdate
	}
	clusterresource.RorMeta.LastReported = time.Now().String()
	clusterresource.Metadata.Name = interregator.GetClusterId()
	clusterresource.KubernetesClusterResource.Status.AgentStatus = rortypes.KubernetesClusterAgentStatus{
		ClusterId:          agentclient.GetClusterId(),
		ClusterName:        interregator.GetClusterName(),
		KubernetesProvider: interregator.GetKubernetesProvider(),
		Az:                 interregator.GetAz(),
		Region:             interregator.GetRegion(),
		Country:            interregator.GetCountry(),
		Workspace:          interregator.GetClusterWorkspace(),
		Datacenter:         interregator.GetDatacenter(),
		Environment:        interregator.GetEnvironment(),
		Versions: map[string]string{
			"RorAgent": rorversion.GetRorVersion().Version,
		},
		LastSeen: time.Now(),
	}

	//stringhelper.PrettyprintStruct(clusterresource)

	clusterresource.GenRorHash()

	resourceCacheInterface.AddResource(clusterresource)

	return nil
}
