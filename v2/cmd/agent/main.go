package main

import (
	"os"
	"os/signal"

	"github.com/NorskHelsenett/ror-agent/v2/internal/agentconfig"
	"github.com/NorskHelsenett/ror-agent/v2/internal/clients"
	"github.com/NorskHelsenett/ror-agent/v2/internal/handlers/dynamicclienthandler"
	"github.com/NorskHelsenett/ror-agent/v2/internal/scheduler"
	"github.com/NorskHelsenett/ror-agent/v2/internal/services/resourceupdatev2"
	"github.com/NorskHelsenett/ror-agent/v2/pkg/clients/dynamicclient"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorclientconfig"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"syscall"

	"github.com/spf13/viper"

	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	_ = "rebuild 9"
	_, _ = maxprocs.Set(maxprocs.Logger(rlog.Infof))
	agentconfig.Init()
	rlog.Info("Agent is starting", rlog.String("version", viper.GetString(configconsts.VERSION)), rlog.String("commit", viper.GetString(configconsts.COMMIT)))
	sigs := make(chan os.Signal, 1)                                    // Create channel to receive os signals
	stop := make(chan struct{})                                        // Create channel to receive stop signal
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	clientConfig := rorclientconfig.ClientConfig{
		Role:                     viper.GetString(configconsts.ROLE),
		Namespace:                viper.GetString(configconsts.POD_NAMESPACE),
		ApiKeySecret:             viper.GetString(configconsts.API_KEY_SECRET),
		ApiKey:                   viper.GetString(configconsts.API_KEY),
		ApiEndpoint:              viper.GetString(configconsts.API_ENDPOINT),
		RorVersion:               agentconfig.GetRorVersion(),
		MustInitializeKubernetes: true,
	}

	clients.InitClients(clientConfig)

	err := resourceupdatev2.ResourceCache.Init()
	if err != nil {
		rlog.Fatal("could not get hashlist for clusterid", err)
	}

	dynamicclienthandler := dynamicclienthandler.NewDynamicClientHandler()
	err = dynamicclient.Start(clients.Kubernetes, dynamicclienthandler, stop, sigs)
	if err != nil {
		rlog.Fatal("could not start dynamic client", err)
	}

	scheduler.SetUpScheduler()

	<-stop
	rlog.Info("Shutting down...")
}
