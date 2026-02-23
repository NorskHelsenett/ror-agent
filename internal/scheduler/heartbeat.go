package scheduler

import (
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/NorskHelsenett/ror-agent/internal/services"
)

func HeartbeatReporting(rorClientInterface clusteragentclient.RorAgentClientInterface) error {
	clusterReport, err := services.GetHeartbeatReport(rorClientInterface)
	if err != nil {
		rlog.Error("error when getting heartbeat report", err)
		return err
	}

	err = rorClientInterface.GetRorClient().V1().Clusters().SendHeartbeat(clusterReport)
	if err != nil {
		rlog.Error("error when sending heartbeat report to ror", err)
		return err
	}
	rlog.Info("heartbeat report sent to ror")
	return nil
}
