package scheduler

import (
	"context"
	"time"

	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	apimachinery "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MetricsReporting(rorAgentClientInterface clusteragentclient.RorAgentClientInterface) error {
	var metricsReport apicontracts.MetricsReport
	rorClientInterface := rorAgentClientInterface.GetRorClient()

	metricsReportNodes, err := CreateNodeMetricsList(rorAgentClientInterface.GetKubernetesClientset())
	if err != nil {
		rlog.Error("error converting podmetrics", err)
		return err
	}
	owner := rorClientInterface.GetOwnerref()
	metricsReport.Owner = apiresourcecontracts.ResourceOwnerReference{
		Scope:   owner.Scope,
		Subject: string(owner.Subject),
	}
	metricsReport.Nodes = metricsReportNodes

	return rorClientInterface.Metrics().PostReport(context.TODO(), metricsReport)

}

func CreateNodeMetricsList(k8sClient *kubernetesclient.K8sClientsets) ([]apicontracts.NodeMetric, error) {

	var metricsReportNodes []apicontracts.NodeMetric
	metricsClient, err := k8sClient.GetMetricsV1Beta1Client()
	if err != nil {
		return metricsReportNodes, err
	}

	nodeMetrics, err := metricsClient.NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		rlog.Error("error getting node metrics", err)
		return metricsReportNodes, err
	}

	// Process directly without additional JSON marshaling/unmarshaling
	for _, node := range nodeMetrics.Items {
		// Convert directly from the typed objects
		metricsReportNode := apicontracts.NodeMetric{
			Name:        node.Name,
			TimeStamp:   node.Timestamp.Time,
			CpuUsage:    node.Usage.Cpu().MilliValue(),
			MemoryUsage: node.Usage.Memory().Value(),
		}
		metricsReportNodes = append(metricsReportNodes, metricsReportNode)
	}

	return metricsReportNodes, nil
}

func CreateNodeMetrics(node apicontracts.NodeMetricsListItem) (apicontracts.NodeMetric, error) {
	var nodeMetric apicontracts.NodeMetric
	var timestamp time.Time = node.Timestamp

	nodeCpuRaw, err := apimachinery.ParseQuantity(node.Usage.CPU)
	if err != nil {
		rlog.Error("error converting nodemetrics", err)
		return nodeMetric, err
	}
	nodeCpu := nodeCpuRaw.MilliValue()

	nodeMemoryRaw, err := apimachinery.ParseQuantity(node.Usage.Memory)
	if err != nil {
		rlog.Error("error converting nodemetrics", err)
		return nodeMetric, err
	}
	nodeMemory, _ := nodeMemoryRaw.AsInt64()

	nodeMetric = apicontracts.NodeMetric{
		Name:        node.Metadata.Name,
		TimeStamp:   timestamp,
		CpuUsage:    nodeCpu,
		MemoryUsage: nodeMemory,
	}
	return nodeMetric, nil
}
