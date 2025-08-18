package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"github.com/NorskHelsenett/ror-agent/v2/internal/agentconfig"
	"github.com/NorskHelsenett/ror-agent/v2/internal/clients"
	"github.com/NorskHelsenett/ror-agent/v2/internal/handlers/clusterhandler"
	"github.com/NorskHelsenett/ror-agent/v2/internal/handlers/dynamicclienthandler"
	"github.com/NorskHelsenett/ror-agent/v2/internal/scheduler"
	"github.com/NorskHelsenett/ror-agent/v2/pkg/clients/dynamicclient"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache"

	healthserver "github.com/NorskHelsenett/ror/pkg/helpers/rorhealth/server"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"syscall"

	"github.com/spf13/viper"

	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	var err error
	_ = "rebuild 12"
	_, _ = maxprocs.Set(maxprocs.Logger(rlog.Infof))
	agentconfig.Init()
	rlog.Info("Agent is starting", rlog.String("version", viper.GetString(configconsts.VERSION)), rlog.String("commit", viper.GetString(configconsts.COMMIT)))
	sigs := make(chan os.Signal, 1)                                    // Create channel to receive os signals
	stop := make(chan struct{})                                        // Create channel to receive stop signal
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	if viper.GetBool(configconsts.ENABLE_PPROF) {
		go func() {
			rlog.Info("Starting pprof server on port 6060")
			err := http.ListenAndServe("localhost:6060", nil) // Start pprof server on localhost:6060
			if err != nil {
				rlog.Fatal("could not start pprof server", err)
			}
		}()
	}

	rlog.Info("Initializing health server")
	_ = healthserver.Start(healthserver.ServerString(viper.GetString(configconsts.HEALTH_ENDPOINT)))

	rorClientInterface, err := clients.NewRorClientInterface()
	if err != nil {
		rlog.Fatal("could not get RorClientInterface", err)
	}

	resourceCache, err := resourcecache.NewResourceCache(resourcecache.ResourceCacheConfig{WorkQueueInterval: 10, RorClient: rorClientInterface})

	if err != nil {
		rlog.Fatal("could not get hashlist for clusterid", err)
	}

	rlog.Info("Starting cluster handler")
	err = clusterhandler.Start()
	if err != nil {
		rlog.Fatal("could not start cluster handler", err)
	}

	rlog.Info("Starting dynamic client handler")
	dynamicclienthandler := dynamicclienthandler.NewDynamicClientHandler(resourceCache)
	err = dynamicclient.Start(rorClientInterface.GetKubernetesClientSet(), dynamicclienthandler, stop, sigs)
	if err != nil {
		rlog.Fatal("could not start dynamic client", err)
	}

	rlog.Info("Starting schedulers")
	scheduler.SetUpScheduler(rorClientInterface)

	<-stop
	rlog.Info("Shutting down...")
}
