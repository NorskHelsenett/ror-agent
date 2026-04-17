package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/common/pkg/config/agentconsts"
	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/rlog"
)

// prometheusQueryResult represents the JSON response from Prometheus instant query API.
type prometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// NodeExporterReporting queries Prometheus for node_exporter metrics and posts a report.
func NodeExporterReporting(rorAgentClientInterface clusteragentclient.RorAgentClientInterface) error {
	prometheusURL := rorconfig.GetString(agentconsts.PrometheusURLEnv)
	if prometheusURL == "" {
		return nil // Prometheus not configured, skip silently
	}

	rorClientInterface := rorAgentClientInterface.GetRorClient()
	owner := rorClientInterface.GetOwnerref()

	nodes, err := collectNodeExporterMetrics(prometheusURL)
	if err != nil {
		rlog.Error("error collecting node_exporter metrics", err)
		return err
	}

	if len(nodes) == 0 {
		return nil
	}

	report := apicontracts.MetricsReport{
		Owner: apiresourcecontracts.ResourceOwnerReference{
			Scope:   owner.Scope,
			Subject: string(owner.Subject),
		},
		Nodes: nodes,
	}

	return rorClientInterface.Metrics().PostReport(context.TODO(), report)
}

func collectNodeExporterMetrics(prometheusURL string) ([]apicontracts.NodeMetric, error) {
	now := time.Now()
	queries := map[string]string{
		"cpu":        `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`,
		"cpu_cores":  `count by (instance) (node_cpu_seconds_total{mode="idle"})`,
		"cpu_used":   `sum by (instance) (rate(node_cpu_seconds_total{mode!="idle"}[5m]))`,
		"mem_used":   `node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes`,
		"mem_total":  `node_memory_MemTotal_bytes`,
		"disk_used":  `sum by (instance) (node_filesystem_size_bytes{fstype!~"tmpfs|overlay"} - node_filesystem_avail_bytes{fstype!~"tmpfs|overlay"})`,
		"disk_total": `sum by (instance) (node_filesystem_size_bytes{fstype!~"tmpfs|overlay"})`,
		"net_rx":     `sum by (instance) (rate(node_network_receive_bytes_total{device!~"lo|veth.*|cali.*|flannel.*"}[5m]))`,
		"net_tx":     `sum by (instance) (rate(node_network_transmit_bytes_total{device!~"lo|veth.*|cali.*|flannel.*"}[5m]))`,
		"load1":      `node_load1`,
		"load5":      `node_load5`,
		"load15":     `node_load15`,
	}

	results := make(map[string]map[string]float64)
	for name, query := range queries {
		values, err := queryPrometheus(prometheusURL, query)
		if err != nil {
			rlog.Warn("prometheus query failed", rlog.String("query", name), rlog.String("error", err.Error()))
			continue
		}
		results[name] = values
	}

	// Collect all unique instance names
	instances := make(map[string]bool)
	for _, vals := range results {
		for inst := range vals {
			instances[inst] = true
		}
	}

	metrics := make([]apicontracts.NodeMetric, 0, len(instances))
	for inst := range instances {
		memTotal := getVal(results, "mem_total", inst)
		memUsed := getVal(results, "mem_used", inst)
		diskTotal := getVal(results, "disk_total", inst)
		diskUsed := getVal(results, "disk_used", inst)

		var memPercent float64
		if memTotal > 0 {
			memPercent = (memUsed / memTotal) * 100
		}
		var diskPercent float64
		if diskTotal > 0 {
			diskPercent = (diskUsed / diskTotal) * 100
		}

		cpuPercent := getVal(results, "cpu", inst)
		cpuCores := getVal(results, "cpu_cores", inst)
		cpuUsedCores := getVal(results, "cpu_used", inst)

		m := apicontracts.NodeMetric{
			Name:             inst,
			TimeStamp:        now,
			CpuUsage:         int64(cpuUsedCores * 1000),
			CpuAllocated:     int64(cpuCores) * 1000,
			CpuPercentage:    cpuPercent,
			MemoryUsage:      int64(memUsed),
			MemoryAllocated:  int64(memTotal),
			MemoryPercentage: memPercent,
			DiskUsageBytes:   int64(diskUsed),
			DiskTotalBytes:   int64(diskTotal),
			DiskPercent:      diskPercent,
			NetworkRxBytes:   getVal(results, "net_rx", inst),
			NetworkTxBytes:   getVal(results, "net_tx", inst),
			Load1:            getVal(results, "load1", inst),
			Load5:            getVal(results, "load5", inst),
			Load15:           getVal(results, "load15", inst),
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func getVal(results map[string]map[string]float64, metric, instance string) float64 {
	if m, ok := results[metric]; ok {
		if v, ok := m[instance]; ok {
			return v
		}
	}
	return 0
}

func queryPrometheus(baseURL, query string) (map[string]float64, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus url: %w", err)
	}
	u.Path = "/api/v1/query"
	u.RawQuery = url.Values{"query": {query}}.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, err
	}

	var result prometheusQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query status: %s", result.Status)
	}

	values := make(map[string]float64)
	for _, r := range result.Data.Result {
		instance := r.Metric["instance"]
		// Strip port from instance name (e.g., "node1:9100" -> "node1")
		if host, err := stripPort(instance); err == nil {
			instance = host
		}
		// Parse the scalar value (second element of the value tuple)
		var valStr string
		if err := json.Unmarshal(r.Value[1], &valStr); err != nil {
			continue
		}
		var val float64
		if _, err := fmt.Sscanf(valStr, "%f", &val); err != nil {
			continue
		}
		values[instance] = val
	}

	return values, nil
}

func stripPort(hostport string) (string, error) {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, err
	}
	return host, nil
}
