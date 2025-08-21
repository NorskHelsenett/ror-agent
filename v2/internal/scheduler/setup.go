package scheduler

import (
	"time"

	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/go-co-op/gocron"
)

func SetUpScheduler(rorClientInterface rorclient.RorClientInterface, kubernetesClientset *kubernetesclient.K8sClientsets) {
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Every(1).Minute().Tag("metrics").Do(MetricsReporting, rorClientInterface, kubernetesClientset)
	if err != nil {
		rlog.Fatal("Could not setup scheduler for metrics", err)
		return
	}
	//scheduler.StartAsync()
}
