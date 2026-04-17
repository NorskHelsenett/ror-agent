package scheduler

import (
	"time"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/go-co-op/gocron"
)

func SetUpScheduler(rorAgentClientInterface clusteragentclient.RorAgentClientInterface) {
	rlog.Info("Starting schedulers")
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Every(5).Minutes().StartImmediately().Tag("node-exporter").Do(NodeExporterReporting, rorAgentClientInterface)

	if err != nil {
		rlog.Error("Could not setup scheduler for node-exporter metrics", err)
	}
	scheduler.StartAsync()
}
