package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/NorskHelsenett/ror-agent/internal/services/authservice"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	apimachinery "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

// TODO: Change to use RorClient
func MetricsReporting(rorClientInterface clusteragentclient.RorAgentClientInterface) error {
	k8sClient, err := rorClientInterface.GetKubernetesClientset().GetKubernetesClientset()
	if err != nil {
		return err
	}
	var metricsReport apicontracts.MetricsReport

	metricsReportNodes, err := CreateNodeMetricsList(k8sClient)
	if err != nil {
		rlog.Error("error converting podmetrics", err)
		return err
	}
	ownerref := authservice.CreateOwnerref()

	metricsReport.Owner = apiresourcecontracts.ResourceOwnerReference{
		Scope:   aclmodels.Acl2Scope(ownerref.Scope),
		Subject: string(ownerref.Subject),
	}
	metricsReport.Nodes = metricsReportNodes

	err = rorClientInterface.GetRorClient().V1().Metrics().PostReport(context.TODO(), metricsReport)
	if err != nil {
		rlog.Error("error when sending metrics report to ror", err)
		return err
	}
	rlog.Info("metrics report sent to ror")
	return nil
}

func CreateNodeMetricsList(k8sClient *kubernetes.Clientset) ([]apicontracts.NodeMetric, error) {
	var nodeMetricsList apicontracts.NodeMetricsList
	var metricsReportNodes []apicontracts.NodeMetric

	data, err := k8sClient.RESTClient().Get().AbsPath("apis/metrics.k8s.io/v1beta1/nodes").DoRaw(context.TODO())
	if err != nil {
		rlog.Error("error converting nodemetrics", err)
		return metricsReportNodes, err
	}

	err = json.Unmarshal(data, &nodeMetricsList)
	if err != nil {
		rlog.Error("error unmarshaling podmetrics", err)
		return metricsReportNodes, err
	}

	for _, node := range nodeMetricsList.Items {

		metricsReportNode, err := CreateNodeMetrics(node)
		if err != nil {
			rlog.Error("error converting podmetrics", err)
			return metricsReportNodes, err
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
