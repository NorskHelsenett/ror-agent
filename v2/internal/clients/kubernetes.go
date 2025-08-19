package clients

import (
	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
)

func MustInitializeKubernetesClient() *kubernetesclient.K8sClientsets {
	kubernetesCli := kubernetesclient.NewK8sClientConfig()
	if kubernetesCli == nil {
		panic("failed to initialize kubernetes client")
	}
	return kubernetesCli
}
