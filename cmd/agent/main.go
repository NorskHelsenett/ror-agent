package main

import (
	"os"
	"os/signal"

	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror-agent/internal/scheduler"
	"github.com/NorskHelsenett/ror-agent/internal/services"
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"
	"github.com/NorskHelsenett/ror/pkg/rorresources/rordefs"

	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/dynamicclient"
	"github.com/NorskHelsenett/ror-agent/pkg/services/healthservice"
	"github.com/NorskHelsenett/ror-agent/pkg/services/pprofservice"

	"github.com/NorskHelsenett/ror/pkg/config/rorversion"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"syscall"
)

func main() {
	config.Init()

	pprofservice.MayStartPprof()

	_ = "rebuild 6"
	rlog.Info("Agent is starting", rlog.String("version", rorversion.GetRorVersion().GetVersion()))
	sigs := make(chan os.Signal, 1)                                    // Create channel to receive os signals
	stop := make(chan struct{})                                        // Create channel to receive stop signal
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	go func() {
		services.GetEgressIp()
	}()

	rorClientInterface, err := clusteragentclient.NewRorAgentClient(clusteragentclient.GetDefaultRorAgentClientConfig())
	if err != nil {
		rlog.Fatal("could not get RorClientInterface", err)
	}
	err = resourceupdate.ResourceCache.Init(rorClientInterface)
	if err != nil {
		rlog.Fatal("could not get hashlist for clusterid", err)
	}

	dynamicclient.Start(rorClientInterface, rordefs.Resourcedefs.GetSchemasByType(rordefs.ApiResourceTypeAgent), stop, sigs)

	scheduler.Start(rorClientInterface)

	healthservice.Start()

	<-stop
	rlog.Info("Shutting down...")
}
