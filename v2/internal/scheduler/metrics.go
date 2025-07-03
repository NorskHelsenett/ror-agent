package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/NorskHelsenett/ror-agent/v2/internal/agentconfig"
	"github.com/NorskHelsenett/ror-agent/v2/internal/clients"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	apimachinery "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MetricsReporting() error {
	var metricsReport apicontracts.MetricsReport

	metricsReportNodes, err := CreateNodeMetricsList(clients.Kubernetes)
	if err != nil {
		rlog.Error("error converting podmetrics", err)
		return err
	}
	owner := clients.RorConfig.CreateOwnerref()
	metricsReport.Owner = apiresourcecontracts.ResourceOwnerReference{
		Scope:   owner.Scope,
		Subject: string(owner.Subject),
	}
	metricsReport.Nodes = metricsReportNodes

	err = sendMetricsToRor(metricsReport)

	return err
}

func sendMetricsToRor(metricsReport apicontracts.MetricsReport) error {
	rorClient, err := clients.GetOrCreateRorClient()
	if err != nil {
		rlog.Error("Could not get ror-api client", err)
		agentconfig.IncreaseErrorCount()
		return err
	}

	url := "/v1/metrics"
	response, err := rorClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metricsReport).
		Post(url)
	if err != nil {
		rlog.Error("Could not send metrics data to ror-api", err)
		agentconfig.IncreaseErrorCount()
		return err
	}

	if response == nil {
		rlog.Error("Response is nil", err)
		agentconfig.IncreaseErrorCount()
		return err
	}

	if !response.IsSuccess() {
		agentconfig.IncreaseErrorCount()
		rlog.Error("Got unsuccessful status code from ror-api", err,
			rlog.Int("status code", response.StatusCode()),
			rlog.Int("error count", agentconfig.ErrorCount))
		return err
	} else {
		agentconfig.ResetErrorCount()
		rlog.Info("Metrics report sent to ror")

		byteReport, err := json.Marshal(metricsReport)
		if err == nil {
			rlog.Debug("", rlog.String("byte report", string(byteReport)))
		}
	}
	return nil
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
