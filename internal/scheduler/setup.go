package scheduler

import (
	"time"

	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/go-co-op/gocron"
)

func SetUpScheduler(rorClientInterface clusteragentclient.RorAgentClientInterface) {
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Every(1).Minute().Tag("heartbeat").Do(HeartbeatReporting, rorClientInterface)
	if err != nil {
		rlog.Fatal("Failed to setup heartbeat schedule", err)
	}

	_, err = scheduler.Every(1).Minute().Tag("metrics").Do(MetricsReporting, rorClientInterface)
	if err != nil {
		rlog.Fatal("Failed to setup metric schedule", err)
	}
	scheduler.StartAsync()
}
