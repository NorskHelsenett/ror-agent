package main

import (
	_ "net/http/pprof"

	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/common/pkg/clients/dynamicclient"
	"github.com/NorskHelsenett/ror-agent/common/pkg/services/healthservice"
	"github.com/NorskHelsenett/ror-agent/common/pkg/services/pprofservice"
	"github.com/NorskHelsenett/ror-agent/v2/internal/agentconfig"
	"github.com/NorskHelsenett/ror-agent/v2/internal/handlers/clusterhandler"
	"github.com/NorskHelsenett/ror-agent/v2/internal/handlers/dynamicclienthandler"
	"github.com/NorskHelsenett/ror-agent/v2/internal/scheduler"

	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"

	"github.com/NorskHelsenett/ror/pkg/rlog"
)

func main() {
	agentconfig.Init()

	pprofservice.MayStartPprof()

	rlog.Info("Agent is starting", rlog.String("version", rorversion.GetRorVersion().GetVersion()), rlog.String("commit", rorversion.GetRorVersion().GetCommit()))

	rorClientInterface := clusteragentclient.MustInitNewRorAgentClient(clusteragentclient.GetDefaultRorAgentClientConfig())

	resourceCache := resourcecache.MustInitNewResourceCache(resourcecache.ResourceCacheConfig{WorkQueueInterval: 10, RorClient: rorClientInterface.GetRorClient()})

	clusterhandler.MustStart(rorClientInterface, resourceCache)

	dynamicclient.MustStart(rorClientInterface, dynamicclienthandler.NewDynamicClientHandler(resourceCache))

	scheduler.SetUpScheduler(rorClientInterface)

	healthservice.MustStart()

	<-rorClientInterface.GetStopChan()
	rlog.Info("Shutting down...")
}
