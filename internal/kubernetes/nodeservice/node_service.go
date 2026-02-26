// TODO: This internal package is copied from ror, should determine if its common and should be moved to ror/pkg
package nodeservice

import (
	"context"
	"math"

	"github.com/NorskHelsenett/ror-agent/internal/kubernetes/k8smodels"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNodes(rorClientInterface clusteragentclient.RorAgentClientInterface) ([]k8smodels.Node, error) {
	var nodes []k8smodels.Node

	metricsClient, err := rorClientInterface.GetKubernetesClientset().GetMetricsClient()
	if err != nil {
		return nodes, err
	}

	for _, node := range rorClientInterface.GetClusterInterregator().Nodes().Get() {

		n := k8smodels.Node{}

		n.Provider = rorClientInterface.GetClusterInterregator().GetMachineProvider()

		n.OsImage = node.Status.NodeInfo.OSImage
		n.Created = node.CreationTimestamp.Time
		n.Annotations = node.Annotations
		n.Name = node.Name
		n.Labels = node.Labels
		n.MachineName = n.Labels["kubernetes.io/hostname"]
		if n.MachineName == "" {
			n.MachineName = node.Name
		}

		n.Architecture = node.Status.NodeInfo.Architecture
		n.ContainerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion
		n.KernelVersion = node.Status.NodeInfo.KernelVersion
		n.KubeProxyVersion = node.Status.NodeInfo.KubeProxyVersion
		n.KubeletVersion = node.Status.NodeInfo.KubeletVersion
		n.OperatingSystem = node.Status.NodeInfo.OperatingSystem

		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(context.TODO(), node.Name, v1.GetOptions{})
		if err == nil {
			cpuUsage := nodeMetrics.Usage.Cpu()
			cpuAllocated, _ := node.Status.Allocatable.Cpu().AsInt64()
			if cpuAllocated == 0 {
				cpudec := node.Status.Allocatable.Cpu().AsDec()
				rounded := cpudec.UnscaledBig().Int64()
				cpuAllocated = int64(math.Round(float64(rounded) / 1000))
			}
			memoryUsageInt, _ := nodeMetrics.Usage.Memory().AsInt64()
			memoryAllocated := node.Status.Allocatable.Memory().Value()

			n.Resources = apicontracts.NodeResources{
				Allocated: apicontracts.ResourceAllocated{
					Cpu:         cpuAllocated,
					MemoryBytes: memoryAllocated,
				},
				Consumed: apicontracts.ResourceConsumed{
					CpuMilliValue: cpuUsage.MilliValue(),
					MemoryBytes:   memoryUsageInt,
				},
			}
		} else {
			rlog.Debug("could not fetch node metrics", rlog.String("name", node.Name))
		}

		nodes = append(nodes, n)
	}

	return nodes, nil
}
