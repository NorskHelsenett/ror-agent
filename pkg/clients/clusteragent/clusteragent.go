package clusteragent

import (
	"context"

	"github.com/NorskHelsenett/ror-agent/internal/checks/initialchecks"
	"github.com/NorskHelsenett/ror-agent/internal/kubernetes/operator/initialize"
	"github.com/NorskHelsenett/ror-agent/internal/services"
	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func InitializeClusterAgent(kubernetesclientset *kubernetesclient.K8sClientsets) error {

	k8sClient, err := kubernetesclientset.GetKubernetesClientset()
	if err != nil {
		panic(err.Error())
	}

	ns := rorconfig.GetString(configconsts.POD_NAMESPACE)
	if ns == "" {
		rlog.Fatal("POD_NAMESPACE is not set", nil)
	}

	_, err = k8sClient.CoreV1().Namespaces().Get(context.Background(), ns, metav1.GetOptions{})
	if err != nil {
		rlog.Fatal("could not get namespace", err)
	}

	err = initialchecks.HasSuccessfullRorApiConnection()
	if err != nil {
		rlog.Fatal("could not connect to ror-api", err)
	}

	err = services.ExtractApikeyOrDie()
	if err != nil {
		rlog.Fatal("could not get or create secret", err)
	}

	clusterId, err := initialize.GetOwnClusterId()
	if err != nil {
		rlog.Fatal("could not fetch clusterid from ror-api", err)
	}
	rorconfig.Set(configconsts.CLUSTER_ID, clusterId)
	return nil
}
