package main

import (
	_ "net/http/pprof"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/common/pkg/services/healthservice"
	"github.com/NorskHelsenett/ror-agent/common/pkg/services/pprofservice"
	"github.com/NorskHelsenett/ror-agent/v2/internal/agentconfig"
	"github.com/NorskHelsenett/ror-agent/v2/internal/scheduler"

	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rordefs"

	"github.com/NorskHelsenett/ror/pkg/rlog"
)

func main() {
	agentconfig.Init()

	pprofservice.MayStartPprof()

	rlog.Info("Agent is starting", rlog.String("version", rorversion.GetRorVersion().GetVersion()), rlog.String("commit", rorversion.GetRorVersion().GetCommit()))

	rorClientInterface := clusteragentclient.MustInitNewRorAgentClient(
		clusteragentclient.WithDynamicClient(rordefs.Resourcedefs.GetSchemasByType(rordefs.ApiResourceTypeClusterAgentV2)...),
	)

	scheduler.SetUpScheduler(rorClientInterface)

	healthservice.MustStart()

	<-rorClientInterface.GetStopChan()
	rlog.Info("Shutting down...")
}
