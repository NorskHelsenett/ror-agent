package scheduler

import (
	"time"

	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/go-co-op/gocron"
)

func SetUpScheduler(rorAgentClientInterface clusteragentclient.RorAgentClientInterface) {
	rlog.Info("Starting schedulers")
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Every(1).Minute().Tag("metrics").Do(MetricsReporting, rorAgentClientInterface)
	if err != nil {
		rlog.Fatal("Could not setup scheduler for metrics", err)
		return
	}
	//scheduler.StartAsync()
}
