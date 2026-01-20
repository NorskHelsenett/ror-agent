package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/NorskHelsenett/ror-agent/internal/clients/clients"
	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror-agent/internal/controllers"
	"github.com/NorskHelsenett/ror-agent/internal/httpserver"
	"github.com/NorskHelsenett/ror-agent/internal/scheduler"
	"github.com/NorskHelsenett/ror-agent/internal/services"
	"github.com/NorskHelsenett/ror-agent/internal/services/resourceupdate"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragent"

	"github.com/NorskHelsenett/ror/pkg/config/rorversion"

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

	go func() {
		services.GetEgressIp()
		sig := <-sigs
		_, _ = fmt.Println()
		_, _ = fmt.Print(sig)
		stop <- struct{}{}
	}()

	clients.Initialize()

	discoveryClient, err := clients.Kubernetes.GetDiscoveryClient()
	if err != nil {
		rlog.Error("failed to get discovery client", err)
	}

	dynamicClient, err := clients.Kubernetes.GetDynamicClient()
	if err != nil {
		rlog.Error("failed to get dynamic client", err)
	}

	// start of refactoring initialize apikey
	// k8sclient, viper

	err = clusteragent.InitializeClusterAgent(clients.Kubernetes)
	if err != nil {
		rlog.Fatal("could not initialize cluster agent", err)
	}

	// end

	err = resourceupdate.ResourceCache.Init()
	if err != nil {
		rlog.Fatal("could not get hashlist for clusterid", err)
	}

	err = scheduler.HeartbeatReporting()
	if err != nil {
		rlog.Fatal("could not send heartbeat to api", err)
	}

	// waiting for ip check to finish :)
	time.Sleep(time.Second * 1)

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

	go func() {
		httpserver.InitHttpServer()
		sig := <-sigs
		_, _ = fmt.Println()
		_, _ = fmt.Println(sig)
		stop <- struct{}{}
	}()

	scheduler.SetUpScheduler()

	<-stop
	rlog.Info("Shutting down...")
}
