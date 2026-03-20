package clusterhandler

import (
	"context"
	"time"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/interregatortypes/v2"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/providers/providermodels"
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

	if err := updateClusterResource(agentclient, resourceCacheInterface); err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := updateClusterResource(agentclient, resourceCacheInterface); err != nil {
				rlog.Error("error updating cluster resource", err)
			}
		}
	}()

	return nil
}

func updateClusterResource(agentclient clusteragentclient.RorAgentClientInterface, resourceCacheInterface resourcecache.ResourceCacheInterface) error {
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
			Environment:        getEnvironment(agentclient, interregator),
			Versions: map[string]string{
				"RorAgent": rorversion.GetRorVersion().Version,
			},
			LastSeen: time.Now(),
		}
	}

	if len(existing.Resources) == 1 {
		clusterresource = rorresources.NewResourceFromStruct(*existing.Resources[0])
		clusterresource.RorMeta.Action = rortypes.K8sActionUpdate
		clusterresource.RorMeta.LastReported = time.Now().String()
		clusterresource.Metadata.Name = interregator.GetClusterId()
		clusterresource.KubernetesClusterResource.Status.AgentStatus = rortypes.KubernetesClusterAgentStatus{
			LastSeen: time.Now(),
		}
	}

	//stringhelper.PrettyprintStruct(clusterresource)

	clusterresource.GenRorHash()

	resourceCacheInterface.AddResource(clusterresource)

	return nil
}

const (
	hintsConfigmap = "nhn-tooling"
)

// getEnvironment determines the environment of the cluster based on the interregator's GetEnvironment method.
// If the interregator returns a known environment, it will try to get a configmap/key
// lastly it will guestimate the environment based on the cluster name, region and az, using a simple heuristic.
func getEnvironment(agentclient clusteragentclient.RorAgentClientInterface, interregator interregatortypes.ClusterInterregator) string {
	interregatorEnv := interregator.GetEnvironment()
	if interregatorEnv != providermodels.UNKNOWN_UNDEFINED && interregatorEnv != providermodels.UNKNOWN_ENVIRONMENT {
		return interregatorEnv
	}
	cmEnv := getEnvironmentFromConfigMap(agentclient)
	if cmEnv != providermodels.UNKNOWN_ENVIRONMENT {
		return cmEnv
	}

	return guessEnvironment(interregator.GetClusterName())
}

func getEnvironmentFromConfigMap(agentclient clusteragentclient.RorAgentClientInterface) string {
	// Try to get environment from configmap
	client, err := agentclient.GetKubernetesClientset().GetKubernetesClientset()
	if err != nil {
		rlog.Warn("could not get kubernetes clientset to get configmap, falling back to heuristic", rlog.String("configmap", hintsConfigmap))
		return providermodels.UNKNOWN_ENVIRONMENT
	}

	cm, err := client.CoreV1().ConfigMaps(rorconfig.GetString(rorconfig.POD_NAMESPACE)).Get(context.TODO(), hintsConfigmap, v1.GetOptions{})
	if err != nil {
		rlog.Warn("could not get configmap for environment hints, falling back to heuristic", rlog.String("configmap", hintsConfigmap), rlog.String("namespace", rorconfig.GetString(rorconfig.POD_NAMESPACE)))
		return providermodels.UNKNOWN_ENVIRONMENT
	}
	env, ok := cm.Data["environment"]
	if !ok {
		rlog.Warn("configmap for environment hints did not contain 'environment' key, falling back to heuristic", rlog.String("configmap", hintsConfigmap))
		return providermodels.UNKNOWN_ENVIRONMENT
	}
	return env
}

func guessEnvironment(clusterName string) string {
	// if the name is [dtqp]-* we assume it's a dev/test/qa/prod cluster, and we use that as environment
	if len(clusterName) > 2 {
		prefix := clusterName[:2]
		switch prefix {
		case "d-":
			return "dev"
		case "t-":
			return "test"
		case "q-":
			return "qa"
		case "p-":
			return "prod"
		}
	}

	return providermodels.UNKNOWN_UNDEFINED
}
