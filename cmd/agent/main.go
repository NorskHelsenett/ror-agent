package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"github.com/NorskHelsenett/ror-agent/internal/clients/clients"
	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror-agent/internal/controllers"
	"github.com/NorskHelsenett/ror-agent/internal/scheduler"
	"github.com/NorskHelsenett/ror-agent/internal/services"
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"

	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"

	healthserver "github.com/NorskHelsenett/ror/pkg/helpers/rorhealth/server"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"syscall"

	"k8s.io/client-go/discovery"
)

func main() {
	config.Init()
	_ = "rebuild 6"
	rlog.Info("Agent is starting", rlog.String("version", rorversion.GetRorVersion().GetVersion()))
	sigs := make(chan os.Signal, 1)                                    // Create channel to receive os signals
	stop := make(chan struct{})                                        // Create channel to receive stop signal
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	if rorconfig.GetBool(configconsts.ENABLE_PPROF) {
		go func() {
			rlog.Info("Starting pprof server on port 6060")
			err := http.ListenAndServe("localhost:6060", nil)
			if err != nil {
				rlog.Fatal("could not start pprof server", err)
			}
		}()
	}

	go func() {
		services.GetEgressIp()
	}()

	rorClientInterface, err := clusteragentclient.NewRorAgentClient(clusteragentclient.GetDefaultRorAgentClientConfig())
	if err != nil {
		rlog.Fatal("could not get RorClientInterface", err)
	}

	discoveryClient, err := rorClientInterface.GetKubernetesClientset().GetDiscoveryClient()
	if err != nil {
		rlog.Error("failed to get discovery client", err)
	}

	dynamicClient, err := rorClientInterface.GetKubernetesClientset().GetDynamicClient()
	if err != nil {
		rlog.Error("failed to get dynamic client", err)
	}

	err = resourceupdate.ResourceCache.Init(rorClientInterface)
	if err != nil {
		rlog.Fatal("could not get hashlist for clusterid", err)
	}

	schemas := clients.InitSchema()

	for _, schema := range schemas {
		check, err := discovery.IsResourceEnabled(discoveryClient, schema)
		if err != nil {
			rlog.Error("Could not query resources from cluster", err)
		}
		if check {
			controller := controllers.NewDynamicController(dynamicClient, schema)

			go func() {
				controller.Run(stop)
				sig := <-sigs
				_, _ = fmt.Println()
				_, _ = fmt.Println(sig)
				stop <- struct{}{}
			}()
		} else {
			errmsg := fmt.Sprintf("Could not register resource %s", schema.Resource)
			rlog.Info(errmsg)
		}
	}

	scheduler.SetUpScheduler(rorClientInterface)

	rlog.Info("Initializing health server")
	err = healthserver.Start(healthserver.ServerString(rorconfig.GetString(configconsts.HEALTH_ENDPOINT)))

	if err != nil {
		rlog.Fatal("could not start health server", err)
	}

	<-stop
	rlog.Info("Shutting down...")
}
