package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	vitiv1alpha1 "github.com/vitistack/common/pkg/v1alpha1"

	"github.com/NorskHelsenett/ror-agent/internal/kubernetes/k8smodels"
	"github.com/NorskHelsenett/ror-agent/internal/kubernetes/nodeservice"
	"github.com/NorskHelsenett/ror-agent/internal/utils"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/kubernetes/metadatahelper"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/providerinterregationreport"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/providers/providermodels"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var MissingConst = "Missing ..."
var caCertAlerted bool = false

type accessGroups struct {
	accessGroups          []string
	readOnlyAccessGroups  []string
	grafanaAdminGroups    []string
	grafanaReadOnlyGroups []string
	argocdAdminGroups     []string
	argocdReadOnlyGroups  []string
}

func (a accessGroups) StringArray() []string {
	var result []string
	for _, group := range a.accessGroups {
		if group != "" {
			groupname := fmt.Sprintf("Cluster Operator - %s", group)
			result = append(result, groupname)
		}
	}
	for _, group := range a.readOnlyAccessGroups {
		if group != "" {
			groupname := fmt.Sprintf("Cluster ReadOnly - %s", group)
			result = append(result, groupname)
		}
	}
	for _, group := range a.grafanaAdminGroups {
		if group != "" {
			groupname := fmt.Sprintf("Grafana Operator - %s", group)
			result = append(result, groupname)
		}
	}
	for _, group := range a.grafanaReadOnlyGroups {
		if group != "" {
			groupname := fmt.Sprintf("Grafana ReadOnly - %s", group)
			result = append(result, groupname)
		}
	}
	for _, group := range a.argocdAdminGroups {
		if group != "" {
			groupname := fmt.Sprintf("ArgoCD Operator - %s", group)
			result = append(result, groupname)
		}
	}
	for _, group := range a.argocdReadOnlyGroups {
		if group != "" {
			groupname := fmt.Sprintf("ArgoCD ReadOnly - %s", group)
			result = append(result, groupname)
		}
	}
	return result
}

func NewAccessGroupsFromData(data map[string]string) accessGroups {
	var accessGroups accessGroups
	if data == nil {
		return accessGroups
	}

	accessGroups.accessGroups = strings.Split(data["accessGroups"], ";")
	accessGroups.readOnlyAccessGroups = strings.Split(data["readOnlyAccessGroups"], ";")
	accessGroups.grafanaAdminGroups = strings.Split(data["grafanaAdminGroups"], ";")
	accessGroups.grafanaReadOnlyGroups = strings.Split(data["grafanaReadOnlyGroups"], ";")
	accessGroups.argocdAdminGroups = strings.Split(data["argocdAdminGroups"], ";")
	accessGroups.argocdReadOnlyGroups = strings.Split(data["argocdReadOnlyGroups"], ";")

	return accessGroups
}

func GetHeartbeatReport(rorClientInterface clusteragentclient.RorAgentClientInterface) (apicontracts.Cluster, error) {

	k8sClient, err := rorClientInterface.GetKubernetesClientset().GetKubernetesClientset()
	if err != nil {
		return apicontracts.Cluster{}, err
	}

	clusterName := "localhost"
	workspaceName := "localhost"
	datacenterName := "local"
	provider := providermodels.ProviderTypeUnknown

	nhnToolingMetadata, err := getNhnToolingMetadata(rorClientInterface)
	if err != nil {
		rlog.Warn("NHN-Tooling is not installed?!")
	}

	kubernetesVersion := getKubernetesServerVersion(rorClientInterface)

	nodes, err := nodeservice.GetNodes(rorClientInterface)
	if err != nil {
		rlog.Error("error getting nodes", err)
	}
	interregationreport, err := providerinterregationreport.GetInterregationReport(rorClientInterface.GetClusterInterregator())

	var k8sControlPlaneEndpoint string = MissingConst
	var controlPlane = apicontracts.ControlPlane{}
	nodePools := make([]apicontracts.NodePool, 0)
	for _, node := range nodes {
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			appendNodeToControlePlane(&node, &controlPlane)
		} else {
			appendNodeToNodePools(&nodePools, &node)
		}
	}

	k8sControlPlaneEndpoint, err = getControlPlaneEndpoint(k8sClient)
	if err != nil {
		rlog.Error("could not get control plane endpoint", err)
	}

	k8sCaCertificate := getCaCertificateFromPod()

	clusterName = interregationreport.ClusterName
	workspaceName = interregationreport.Workspace
	datacenterName = interregationreport.Datacenter
	provider = interregationreport.KubernetesProvider

	ingresses, err := getIngresses(k8sClient)
	if err != nil {
		rlog.Error("could not get ingresses", err)
	}

	nodeCount := int64(0)
	cpuSum := int64(0)
	cpuConsumedSum := int64(0)
	memorySum := int64(0)
	memoryConsumedSum := int64(0)
	for i := 0; i < len(nodePools); i++ {
		nodepool := nodePools[i]
		nodeCount = nodeCount + nodepool.Metrics.NodeCount
		cpuSum = cpuSum + nodepool.Metrics.Cpu
		cpuConsumedSum = cpuConsumedSum + nodepool.Metrics.CpuConsumed
		memorySum = memorySum + nodepool.Metrics.Memory
		memoryConsumedSum = memoryConsumedSum + nodepool.Metrics.MemoryConsumed
	}

	agentVersion := rorversion.GetRorVersion().GetVersion()
	agentSha := rorversion.GetRorVersion().GetCommit()

	var created time.Time
	kubeSystem := "kube-system"
	kubeSystemNamespace, err := k8sClient.CoreV1().Namespaces().Get(context.Background(), kubeSystem, v1.GetOptions{})
	if err != nil {
		rlog.Error("could not fetch namespace", err, rlog.String("namespace", kubeSystem))
	} else {
		created = kubeSystemNamespace.CreationTimestamp.Time
	}

	report := apicontracts.Cluster{
		ACL: apicontracts.AccessControlList{
			AccessGroups: nhnToolingMetadata.AccessGroups,
		},
		Environment: nhnToolingMetadata.Environment,
		ClusterId:   rorconfig.GetString(configconsts.CLUSTER_ID),
		ClusterName: clusterName,
		Ingresses:   ingresses,
		Created:     created,
		Topology: apicontracts.Topology{
			ControlPlaneEndpoint: k8sControlPlaneEndpoint,
			EgressIp:             EgressIp,
			ControlPlane:         controlPlane,
			NodePools:            nodePools,
		},
		Versions: apicontracts.Versions{
			Kubernetes: kubernetesVersion,
			NhnTooling: apicontracts.NhnTooling{
				Version:     nhnToolingMetadata.Version,
				Branch:      nhnToolingMetadata.Branch,
				Environment: nhnToolingMetadata.Environment,
			},
			Agent: apicontracts.Agent{
				Version: agentVersion,
				Sha:     agentSha,
			},
		},
		Metrics: apicontracts.Metrics{
			NodeCount:      nodeCount,
			NodePoolCount:  int64(len(nodePools)),
			Cpu:            cpuSum,
			CpuConsumed:    cpuConsumedSum,
			Memory:         memorySum,
			MemoryConsumed: memoryConsumedSum,
			ClusterCount:   1,
		},
		Workspace: apicontracts.Workspace{
			Name: workspaceName,
			Datacenter: apicontracts.Datacenter{
				Name:     datacenterName,
				Provider: provider,
				Location: apicontracts.DatacenterLocation{
					Region:  interregationreport.Region,
					Country: interregationreport.Country,
				},
			},
		},
		KubeApi: apicontracts.ClusterKubeApi{
			EndpointAddress: k8sControlPlaneEndpoint,
			Certificate:     k8sCaCertificate,
		},
	}
	return report, nil
}

func getCaCertificateFromPod() string {
	var k8sCaCertificate string

	// Get the cluster CA certificate from the service account's ca.crt file
	// This is the standard way to get the CA certificate when running inside a pod
	caCertPath := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	if caCertData, err := os.ReadFile(caCertPath); err == nil {
		k8sCaCertificate = base64.StdEncoding.EncodeToString(caCertData)
		rlog.Debug("Successfully read cluster CA certificate from service account")
	} else {
		// Fallback: try to get it from REST config using the config package

		if restConfig, err := rest.InClusterConfig(); err == nil {
			rlog.Warn("Could not read CA certificate from service account", rlog.String("path", caCertPath), rlog.Any("error", err))
			if len(restConfig.CAData) > 0 {
				k8sCaCertificate = base64.StdEncoding.EncodeToString(restConfig.CAData)
				rlog.Debug("Successfully read cluster CA certificate from REST config CAData")
			} else if restConfig.CAFile != "" {
				if caCertData, err := os.ReadFile(restConfig.CAFile); err == nil {
					k8sCaCertificate = base64.StdEncoding.EncodeToString(caCertData)
					rlog.Debug("Successfully read cluster CA certificate from REST config CAFile")
				} else {
					rlog.Error("Could not read CA certificate file", err, rlog.String("caFile", restConfig.CAFile))
				}
			}
		} else {
			if !caCertAlerted {
				rlog.Warn("Could not get in-cluster config for CA certificate extraction")
				caCertAlerted = true
			}
		}
	}
	return k8sCaCertificate
}

func getKubernetesServerVersion(rorClientInterface clusteragentclient.RorAgentClientInterface) string {

	client, err := rorClientInterface.GetKubernetesClientset().GetDiscoveryClient()
	if err != nil {
		rlog.Error("could not get discovery client", err)
		return MissingConst
	}

	k8sVersion, err := client.ServerVersion()
	if err != nil {
		rlog.Error("could not get kubernetes server version", err)
	}

	kubernetesVersion := MissingConst
	if k8sVersion != nil {
		kubernetesVersion = k8sVersion.String()
	}

	k8sVersionArray := strings.Split(kubernetesVersion, "+")
	if len(k8sVersionArray) > 1 {
		kubernetesVersion = k8sVersionArray[0]
	}
	return kubernetesVersion
}

func getIngresses(k8sClient *kubernetes.Clientset) ([]apicontracts.Ingress, error) {
	var ingressList []apicontracts.Ingress
	nsList, err := k8sClient.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		rlog.Error("could not fetch namespaces", err)
		return ingressList, errors.New("could not fetch namespaces from cluster")
	}

	for _, namespace := range nsList.Items {
		ing := k8sClient.NetworkingV1().Ingresses(namespace.Name)
		ingresses, err := ing.List(context.Background(), v1.ListOptions{})
		if err != nil {
			rlog.Error("could not list ingress in namespace", err, rlog.String("namespace", namespace.Name))
			continue
		}
		for _, ingress := range ingresses.Items {
			richIngress, err := utils.GetIngressDetails(context.Background(), k8sClient, &ingress)
			if err != nil {
				rlog.Error("could not enrich ingress", err,
					rlog.String("ingress", ingress.Name),
					rlog.String("namespace", namespace.Name))
				continue
			} else {
				ingressList = append(ingressList, *richIngress)
			}
		}
	}

	return ingressList, nil
}

func appendNodeToControlePlane(node *k8smodels.Node, controlPlane *apicontracts.ControlPlane) {
	apiNode := apicontracts.Node{
		Name:                    node.Name,
		Role:                    "control-plane",
		Created:                 node.Created,
		OsImage:                 node.OsImage,
		MachineName:             node.MachineName,
		Architecture:            node.Architecture,
		ContainerRuntimeVersion: node.ContainerRuntimeVersion,
		KernelVersion:           node.KernelVersion,
		KubeProxyVersion:        node.KubeProxyVersion,
		KubeletVersion:          node.KubeletVersion,
		OperatingSystem:         node.OperatingSystem,
		Metrics: apicontracts.Metrics{
			Cpu:            node.Resources.Allocated.Cpu,
			Memory:         node.Resources.Allocated.MemoryBytes,
			CpuConsumed:    node.Resources.Consumed.CpuMilliValue,
			MemoryConsumed: node.Resources.Consumed.MemoryBytes,
		},
	}

	controlPlane.Nodes = append(controlPlane.Nodes, apiNode)

	controlPlane.Metrics.NodeCount = int64(len(controlPlane.Nodes))
	controlPlane.Metrics.Cpu = controlPlane.Metrics.Cpu + apiNode.Metrics.Cpu
	controlPlane.Metrics.Memory = controlPlane.Metrics.Memory + apiNode.Metrics.Memory
	controlPlane.Metrics.CpuConsumed = controlPlane.Metrics.CpuConsumed + apiNode.Metrics.CpuConsumed
	controlPlane.Metrics.MemoryConsumed = controlPlane.Metrics.MemoryConsumed + apiNode.Metrics.MemoryConsumed

}

func appendNodeToNodePools(nodePools *[]apicontracts.NodePool, node *k8smodels.Node) {
	clusterNameSplit := strings.Split(node.ClusterName, "-")
	machineNameSplit := strings.Split(node.MachineName, "-")
	var workerName string
	if node.Provider == providermodels.ProviderTypeTalos {
		workerName = node.Annotations["ror.io/node-pool"]
	} else if node.Provider != providermodels.ProviderTypeAks {
		workerName = machineNameSplit[len(clusterNameSplit)]
	} else {
		workerName = machineNameSplit[1]
	}

	apiNode := apicontracts.Node{
		Role:                    "worker",
		Name:                    node.Name,
		Created:                 node.Created,
		OsImage:                 node.OsImage,
		MachineName:             node.MachineName,
		Architecture:            node.Architecture,
		ContainerRuntimeVersion: node.ContainerRuntimeVersion,
		KernelVersion:           node.KernelVersion,
		KubeProxyVersion:        node.KubeProxyVersion,
		KubeletVersion:          node.KubeletVersion,
		OperatingSystem:         node.OperatingSystem,

		Metrics: apicontracts.Metrics{
			Cpu:            node.Resources.Allocated.Cpu,
			CpuConsumed:    node.Resources.Consumed.CpuMilliValue,
			Memory:         node.Resources.Allocated.MemoryBytes,
			MemoryConsumed: node.Resources.Consumed.MemoryBytes,
		},
	}

	var nodePool *apicontracts.NodePool = nil
	var index int
	for i := 0; i < len(*nodePools); i++ {
		nodepool := (*nodePools)[i]
		if nodepool.Name == workerName {
			index = i
			nodePool = &nodepool
		}
	}

	if nodePool == nil {
		list := []apicontracts.Node{apiNode}
		np := apicontracts.NodePool{
			Name:  workerName,
			Nodes: list,
			Metrics: apicontracts.Metrics{
				NodeCount:      int64(len(list)),
				Cpu:            apiNode.Metrics.Cpu,
				Memory:         apiNode.Metrics.Memory,
				CpuConsumed:    apiNode.Metrics.CpuConsumed,
				MemoryConsumed: apiNode.Metrics.MemoryConsumed,
			},
		}
		*nodePools = append(*nodePools, np)
	} else {
		nodelist := append(nodePool.Nodes, apiNode)
		nodePool.Nodes = nodelist
		nodePool.Metrics.Cpu = nodePool.Metrics.Cpu + apiNode.Metrics.Cpu
		nodePool.Metrics.Memory = nodePool.Metrics.Memory + apiNode.Metrics.Memory
		nodePool.Metrics.CpuConsumed = nodePool.Metrics.CpuConsumed + apiNode.Metrics.CpuConsumed
		nodePool.Metrics.MemoryConsumed = nodePool.Metrics.MemoryConsumed + apiNode.Metrics.MemoryConsumed
		nodePool.Metrics.NodeCount = int64(len(nodelist))
		(*nodePools)[index] = *nodePool
	}
}

func getControlPlaneEndpoint(clientset *kubernetes.Clientset) (string, error) {

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})

	if err != nil {
		errMsg := "getControlPlaneEndpoint: Could not get nodes from k8s"
		return "", errors.New(errMsg)
	}
	for _, node := range nodes.Items {
		if endpoint, ok := metadatahelper.GetAnnotationOrLabel(node.ObjectMeta, vitiv1alpha1.K8sEndpointAnnotation); ok {
			return endpoint, nil
		}
		if endpoint, ok := metadatahelper.GetAnnotationOrLabel(node.ObjectMeta, "ror.io/api-endpoint-addr"); ok {
			return endpoint, nil
		}
	}

	kubeadmConfigMap, err := clientset.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kubeadm-config", v1.GetOptions{})
	if err != nil {
		errMsg := "getControlPlaneEndpoint: Could not get cluster config from kube-system/kubeadm-config, check rbac"
		return "", errors.New(errMsg)
	}

	if kubeadmConfigMap == nil {
		errMsg := "getControlPlaneEndpoint: get value 'ControlPlaneEndpoint' from yaml"
		return "", errors.New(errMsg)
	}

	kubeadmClusterConfiguration := kubeadmConfigMap.Data["ClusterConfiguration"]

	var clusterConfigurationValues K8sClusterConfiguration
	err = yaml.Unmarshal([]byte(kubeadmClusterConfiguration), &clusterConfigurationValues)
	if err != nil {
		errMsg := "getControlPlaneEndpoint: Could not parse yaml string to stuct"
		rlog.Error(errMsg, err)
		return "", errors.New(errMsg)
	}
	if clusterConfigurationValues.ControlPlaneEndpoint == "" {
		errMsg := "getControlPlaneEndpoint: ControlPlaneEndpoint is empty in configmap"
		return "", errors.New(errMsg)
	}
	return clusterConfigurationValues.ControlPlaneEndpoint, nil
}

type K8sClusterConfiguration struct {
	ControlPlaneEndpoint string `yaml:"controlPlaneEndpoint"`
}
type NhnToolingValues struct {
	Cluster NhnToolingCluster `yaml:"cluster"`
	NHN     NHN               `yaml:"nhn"`
}

type NhnToolingCluster struct {
	AccessGroups []string `yaml:"accessGroups"`
}

type NHN struct {
	AccessGroups   []string `yaml:"accessGroups"`
	ToolingVersion string   `yaml:"toolingVersion"`
	Environment    string   `yaml:"environment"`
}
