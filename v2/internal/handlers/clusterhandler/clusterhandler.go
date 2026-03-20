package clusterhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"
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
	if err != nil {
		return fmt.Errorf("error fetching existing resources for cluster handler: %w", err)
	}
	if len(existing.Resources) > 1 {
		rlog.Warn("multiple existing resources for cluster handler, using first", rlog.Int("count", len(existing.Resources)))
		existing.Resources = existing.Resources[:1]
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

	// Full update if resource is new or agent version changed, otherwise just update partial
	needsFullUpdate := len(existing.Resources) == 0 ||
		clusterresource.KubernetesClusterResource.Status.AgentStatus.Versions["RorAgent"] != rorversion.GetRorVersion().Version

	clusterresource.RorMeta.LastReported = time.Now().String()
	clusterresource.Metadata.Name = interregator.GetClusterId()

	hintsData := getHintsConfigMap(agentclient)

	if needsFullUpdate {
		clusterresource.KubernetesClusterResource.Status.AgentStatus = rortypes.KubernetesClusterAgentStatus{
			ClusterId:          agentclient.GetClusterId(),
			ClusterName:        interregator.GetClusterName(),
			KubernetesProvider: interregator.GetKubernetesProvider(),
			Az:                 interregator.GetAz(),
			Region:             interregator.GetRegion(),
			Country:            interregator.GetCountry(),
			Workspace:          interregator.GetClusterWorkspace(),
			Datacenter:         interregator.GetDatacenter(),
			Environment:        getEnvironment(agentclient, hintsData),
			Versions:           getVersions(hintsData),
			Nodes:              getNodes(agentclient),
			LastSeen:           time.Now(),
			CreatedAt:          getCreatedTime(agentclient),
		}
	} else {
		clusterresource.KubernetesClusterResource.Status.AgentStatus.LastSeen = time.Now()
		clusterresource.KubernetesClusterResource.Status.AgentStatus.Versions = getVersions(hintsData)
	}

	//stringhelper.PrettyprintStruct(clusterresource)

	clusterresource.GenRorHash()

	resourceCacheInterface.AddResource(clusterresource)

	return nil
}

func getCreatedTime(agentclient clusteragentclient.RorAgentClientInterface) time.Time {
	// get the kube-system namespace creation time, as a proxy for cluster creation time, as the agent will be deployed shortly after cluster creation
	client, err := agentclient.GetKubernetesClientset().GetKubernetesClientset()
	if err != nil {
		rlog.Warn("could not get kubernetes clientset to get cluster creation time")
		return time.Time{}
	}
	namespace, err := client.CoreV1().Namespaces().Get(context.TODO(), "kube-system", v1.GetOptions{})
	if err != nil {
		rlog.Warn("could not get kube-system namespace to get cluster creation time")
		return time.Time{}
	}
	return namespace.CreationTimestamp.Time
}

func getVersions(hintsData map[string]string) map[string]string {
	return map[string]string{
		"RorAgent":   rorversion.GetRorVersion().Version,
		"NhnTooling": getConfigMapValue(hintsData, "toolingVersion", "Unknown"),
	}
}

func getNodes(agentclient clusteragentclient.RorAgentClientInterface) rortypes.KubernetesClusterAgentStatusNodes {
	interregator := agentclient.GetClusterInterregator()
	nodes := interregator.Nodes().Get()

	nodepoolMap := make(map[string][]rortypes.KubernetesClusterAgentStatusNodesNodepoolsNodes)
	var controlPlane []rortypes.KubernetesClusterAgentStatusNodesNodepoolsNodes

	for _, node := range nodes {
		cpuQuantity := node.Status.Capacity["cpu"]
		memoryQuantity := node.Status.Capacity["memory"]

		nodeInfo := rortypes.KubernetesClusterAgentStatusNodesNodepoolsNodes{
			Name:              node.Name,
			Cpu:               int(cpuQuantity.Value()),
			Memory:            memoryQuantity.Value(),
			Architecture:      node.Status.NodeInfo.Architecture,
			KubernetesVersion: node.Status.NodeInfo.KubeletVersion,
		}

		if _, isControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; isControlPlane {
			controlPlane = append(controlPlane, nodeInfo)
			continue
		}

		poolName := node.Labels["topology.kubernetes.io/zone"]
		if np, ok := node.Labels["node.kubernetes.io/nodepool"]; ok {
			poolName = np
		}
		if poolName == "" {
			poolName = "default"
		}

		nodepoolMap[poolName] = append(nodepoolMap[poolName], nodeInfo)
	}

	var nodepools []rortypes.KubernetesClusterAgentStatusNodesNodepools
	for name, poolNodes := range nodepoolMap {
		nodepools = append(nodepools, rortypes.KubernetesClusterAgentStatusNodesNodepools{
			Name:  name,
			Nodes: poolNodes,
		})
	}

	return rortypes.KubernetesClusterAgentStatusNodes{
		ControllPlane: controlPlane,
		Nodepools:     nodepools,
	}
}

const (
	hintsConfigmap = "nhn-tooling"
)

// getEnvironment determines the environment of the cluster based on the interregator's GetEnvironment method.
// If the interregator returns a known environment, it will try to get a configmap/key
// lastly it will guestimate the environment based on the cluster name, region and az, using a simple heuristic.
func getEnvironment(agentclient clusteragentclient.RorAgentClientInterface, hintsData map[string]string) string {
	interregator := agentclient.GetClusterInterregator()
	interregatorEnv := interregator.GetEnvironment()
	if interregatorEnv != providermodels.UNKNOWN_UNDEFINED && interregatorEnv != providermodels.UNKNOWN_ENVIRONMENT {
		return interregatorEnv
	}

	if env := getConfigMapValue(hintsData, "environment", ""); env != "" {
		return env
	}

	return guessEnvironment(interregator.GetClusterName())
}

// getHintsConfigMap fetches the nhn-tooling configmap, returning nil if unavailable.
func getHintsConfigMap(agentclient clusteragentclient.RorAgentClientInterface) map[string]string {
	client, err := agentclient.GetKubernetesClientset().GetKubernetesClientset()
	if err != nil {
		rlog.Warn("could not get kubernetes clientset to get configmap", rlog.String("configmap", hintsConfigmap))
		return nil
	}
	cm, err := client.CoreV1().ConfigMaps(rorconfig.GetString(rorconfig.POD_NAMESPACE)).Get(context.TODO(), hintsConfigmap, v1.GetOptions{})
	if err != nil {
		rlog.Warn("could not get configmap", rlog.String("configmap", hintsConfigmap), rlog.String("namespace", rorconfig.GetString(rorconfig.POD_NAMESPACE)))
		return nil
	}
	return cm.Data
}

// getConfigMapValue returns the value for a key from a configmap data map, or fallback if missing.
func getConfigMapValue(data map[string]string, key string, fallback string) string {
	if data == nil {
		return fallback
	}
	if val, ok := data[key]; ok {
		return val
	}
	return fallback
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
