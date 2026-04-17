package scheduler

import (
	"time"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/go-co-op/gocron"
)

func MustStart(rorClientInterface clusteragentclient.RorAgentClientInterface) {
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Every(1).Minute().Tag("heartbeat").Do(HeartbeatReporting, rorClientInterface)
	if err != nil {
		rlog.Fatal("Failed to setup heartbeat schedule", err)
	}

	// Metrics reporting is handled by agent v2
	scheduler.StartAsync()
}
