// The package implements clients for the ror-agent
package clients

import (
	"fmt"

	"github.com/NorskHelsenett/ror-agent/internal/config"
	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpauthprovider"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpclient"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/helpers/rorclientconfig"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels/rorresourceowner"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/spf13/viper"
)

type RorAgentClientInterface struct {
	rorAPIClient *rorclient.RorClient
	k8sClientSet *kubernetesclient.K8sClientsets
	ownerRef     rorresourceowner.RorResourceOwnerReference
}

func SetupRORClient() (*rorclient.RorClient, error) {
	//TODO: get apikey from secret/create
	if viper.Get(configconsts.API_KEY) == "" {
		return nil, fmt.Errorf("API_KEY is not set in the configuration")
	}
	role := viper.GetString(configconsts.ROLE)
	authProvider := httpauthprovider.NewAuthProvider(httpauthprovider.AuthPoviderTypeAPIKey, viper.GetString(configconsts.API_KEY))
	clientConfig := httpclient.HttpTransportClientConfig{
		BaseURL:      viper.GetString(configconsts.API_ENDPOINT),
		AuthProvider: authProvider,
		Version:      config.GetRorVersion(),
		Role:         role,
	}
	transport := resttransport.NewRorHttpTransport(&clientConfig)
	rorclient := rorclient.NewRorClient(transport)
	if err := rorclient.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping RorClient: %w", err)
	}

	return rorclient, nil
}

func GetRorClientInterface() (rorclientconfig.RorClientInterface, error) {
	var err error

	// Create a pointer to the struct
	rorClient := &RorAgentClientInterface{}

	rorClient.rorAPIClient, err = SetupRORClient()
	if err != nil {
		rlog.Error("failed to setup RorClient", err)
		return nil, err
	}

	return rorClient, nil
}

func (r *RorAgentClientInterface) GetRorClient() rorclient.RorClientInterface {
	return r.rorAPIClient
}

func (r *RorAgentClientInterface) GetKubernetesClientSet() *kubernetesclient.K8sClientsets {
	return r.k8sClientSet
}

func (r *RorAgentClientInterface) GetOwnerref() rorresourceowner.RorResourceOwnerReference {
	return r.ownerRef
}

func (r *RorAgentClientInterface) SetOwnerref(ownerref rorresourceowner.RorResourceOwnerReference) {
	r.ownerRef = ownerref
}
