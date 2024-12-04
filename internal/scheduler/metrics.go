package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/NorskHelsenett/ror-agent/internal/clients/clients"
	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror-agent/internal/services/authservice"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	apimachinery "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

// TODO: Change to use RorClient
func MetricsReporting() error {
	k8sClient, err := clients.Kubernetes.GetKubernetesClientset()
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

	err = sendMetricsToRor(metricsReport)

	return err
}

func sendMetricsToRor(metricsReport apicontracts.MetricsReport) error {
	rorClient, err := clients.GetOrCreateRorClient()
	if err != nil {
		rlog.Error("Could not get ror-api client", err)
		config.IncreaseErrorCount()
		return err
	}

	url := "/v1/metrics"
	response, err := rorClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metricsReport).
		Post(url)
	if err != nil {
		rlog.Error("Could not send metrics data to ror-api", err)
		config.IncreaseErrorCount()
		return err
	}

	if response == nil {
		rlog.Error("Response is nil", err)
		config.IncreaseErrorCount()
		return err
	}

	if !response.IsSuccess() {
		config.IncreaseErrorCount()
		rlog.Error("Got unsuccessful status code from ror-api", err,
			rlog.Int("status code", response.StatusCode()),
			rlog.Int("error count", config.ErrorCount))
		return err
	} else {
		config.ResetErrorCount()
		rlog.Info("Metrics report sent to ror")

		byteReport, err := json.Marshal(metricsReport)
		if err == nil {
			rlog.Debug("", rlog.String("byte report", string(byteReport)))
		}
	}
	return nil
}

func CreateNodeMetricsList(k8sClient *kubernetes.Clientset) ([]apicontracts.NodeMetric, error) {
	var nodeMetricsList apicontracts.NodeMetricsList
	var metricsReportNodes []apicontracts.NodeMetric

	data, err := k8sClient.RESTClient().Get().AbsPath("apis/metrics.k8s.io/v1beta1/nodes").DoRaw(context.TODO())
	if err != nil {
		rlog.Error("error converting podmetrics", err)
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
