package main

import (
	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror-agent/internal/handlers/dynamichandler"
	"github.com/NorskHelsenett/ror-agent/internal/scheduler"
	"github.com/NorskHelsenett/ror-agent/internal/services"
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"

	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/dynamicclient"
	"github.com/NorskHelsenett/ror-agent/pkg/services/healthservice"
	"github.com/NorskHelsenett/ror-agent/pkg/services/pprofservice"

	"github.com/NorskHelsenett/ror/pkg/config/rorversion"

	"github.com/NorskHelsenett/ror/pkg/rlog"
)

func main() {
	config.Init()

	pprofservice.MayStartPprof()

	rlog.Info("Agent is starting", rlog.String("version", rorversion.GetRorVersion().GetVersion()))

	services.GetEgressIp()

	rorClientInterface := clusteragentclient.MustInitNewRorAgentClient(clusteragentclient.GetDefaultRorAgentClientConfig())

	resourceupdate.ResourceCache.MustInit(rorClientInterface)

	dynamicclient.MustStart(rorClientInterface, dynamichandler.NewDynamicClientHandler())

	scheduler.MustStart(rorClientInterface)

	healthservice.MustStart()

	<-rorClientInterface.GetStopChan()
	rlog.Info("Shutting down...")
}
