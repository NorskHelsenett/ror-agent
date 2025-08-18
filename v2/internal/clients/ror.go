// The package implements clients for the ror-agent
package clients

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NorskHelsenett/ror-agent/internal/config"
	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/clients/kubernetes/clusterinterregator"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpauthprovider"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpclient"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/rorclientconfig"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels/rorresourceowner"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const ERR_SECRET_NOT_FOUND = "___secret_not_found___"

type RorAgentClientInterface struct {
	rorAPIClient *rorclient.RorClient
	k8sClientSet *kubernetesclient.K8sClientsets
	ownerRef     rorresourceowner.RorResourceOwnerReference
	config       ClientConfig
}

type ClientConfig struct {
	role         string
	namespace    string
	apiEndpoint  string
	apiKey       string
	apiKeySecret string
	rorVersion   rorversion.RorVersion
}

func NewRorClientInterface() (rorclientconfig.RorClientInterface, error) {
	var err error

	// Create a pointer to the struct
	rorClient := &RorAgentClientInterface{}

	rorClient.getClientConfig()

	rorClient.k8sClientSet = MustInitializeKubernetesClient()

	// check if kubernetes client is initialized
	if rorClient.k8sClientSet != nil {
		rlog.Debug("Using kubernetes secret to get api-key")
		rorClient.kubernetesAuth()
	}

	err = rorClient.initKubernetesClusterSetup()
	if err != nil {
		rlog.Error("failed to initialize kubernetes cluster setup", err)
		return nil, err
	}

	err = rorClient.initAuthorizedRorClient()
	if err != nil {
		rlog.Error("failed to setup RorClient", err)
		return nil, err
	}

	ver, err := rorClient.rorAPIClient.Info().GetVersion()
	if err != nil {
		return nil, err
	}

	selfdata, err := rorClient.rorAPIClient.Clusters().GetSelf()
	if err != nil {
		return nil, err
	}

	rlog.Info("connected to ror-api", rlog.String("version", ver), rlog.String("clusterid", selfdata.ClusterId))
	rorClient.SetOwnerref(rorresourceowner.RorResourceOwnerReference{
		Scope:   aclmodels.Acl2ScopeCluster,
		Subject: aclmodels.Acl2Subject(selfdata.ClusterId),
	})

	return rorClient, nil
}

func (r *RorAgentClientInterface) getClientConfig() {
	viper.SetDefault(configconsts.API_KEY, ERR_SECRET_NOT_FOUND)

	r.config = ClientConfig{
		role:         viper.GetString(configconsts.ROLE),
		namespace:    viper.GetString(configconsts.POD_NAMESPACE),
		apiKeySecret: viper.GetString(configconsts.API_KEY_SECRET),
		apiKey:       viper.GetString(configconsts.API_KEY),
		apiEndpoint:  viper.GetString(configconsts.API_ENDPOINT),
		rorVersion:   config.GetRorVersion(),
	}

}

func (r *RorAgentClientInterface) initKubernetesClusterSetup() error {

	// check if namespace is set and accessible
	if r.config.namespace == "" {
		return fmt.Errorf("failed to get namespace")
	}
	_, err := r.k8sClientSet.GetNamespace(r.config.namespace)
	if err != nil {
		return fmt.Errorf("failed to get namespace %s", err)
	}

	// check if api endpoint is set and accessible
	if r.config.apiEndpoint == "" {
		return fmt.Errorf("failed to get api endpoint")
	}
	if r.config.apiKey == ERR_SECRET_NOT_FOUND {
		rlog.Info("api key secret not found, interregating cluster and registering new key")

		interregationreport, err := clusterinterregator.InterregateCluster(r.k8sClientSet)

		if err != nil {
			return fmt.Errorf("failed to interregate cluster %s", err)
		}

		r.initUnathorizedRorClient()
		key, err := r.rorAPIClient.Clusters().Register(apicontracts.AgentApiKeyModel{
			Identifier:     interregationreport.ClusterName,
			DatacenterName: interregationreport.Datacenter,
			WorkspaceName:  interregationreport.Workspace,
			Provider:       interregationreport.Provider,
			Type:           "Cluster",
		})
		if err != nil {
			return fmt.Errorf("failed to register cluster %s", err)
		}
		err = r.kubernetesCreateApiKeySecret(key)
		if err != nil {
			return fmt.Errorf("failed to create api key secret %s", err)

		}
	}

	return nil
}

func (r *RorAgentClientInterface) kubernetesCreateApiKeySecret(apiKey string) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.config.apiKeySecret,
			Namespace: r.config.namespace,
		},
		Type: v1.SecretTypeOpaque,
		StringData: map[string]string{
			"APIKEY": apiKey,
		},
	}
	_, err := r.k8sClientSet.CreateSecret(r.config.namespace, secret)
	if err != nil {
		rlog.Error("failed to create api key secret", err)
		return err
	}
	r.config.apiKey = apiKey
	return nil

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

// kubernetesAuth sets the api key from the secret defined in the config value apiKeySecret
// in the namespace defined in the config value namespace
// if the secret does not exist, it will be created
func (r *RorAgentClientInterface) kubernetesAuth() {

	secret, err := r.k8sClientSet.GetSecret(r.config.namespace, r.config.apiKeySecret)
	if err != nil {
		if errors.IsNotFound(err) {
			rlog.Warn("api key secret not found")
			return
		} else {
			rlog.Error("failed to get api key secret", err)
			return
		}
	}
	r.config.apiKey = string(secret.Data["APIKEY"])
}

func (r *RorAgentClientInterface) initUnathorizedRorClient() {
	httptransportconfig := httpclient.HttpTransportClientConfig{
		BaseURL:      r.config.apiEndpoint,
		AuthProvider: httpauthprovider.NewNoAuthprovider(),
		Role:         r.config.role,
		Version:      r.config.rorVersion,
	}
	rorclienttransport := resttransport.NewRorHttpTransport(&httptransportconfig)
	r.rorAPIClient = rorclient.NewRorClient(rorclienttransport)
}
func (r *RorAgentClientInterface) initAuthorizedRorClient() error {

	if r.config.apiKey == ERR_SECRET_NOT_FOUND {
		return fmt.Errorf("API_KEY is not set in the configuration")
	}
	authProvider := httpauthprovider.NewAuthProvider(httpauthprovider.AuthPoviderTypeAPIKey, r.config.apiKey)
	clientConfig := httpclient.HttpTransportClientConfig{
		BaseURL:      r.config.apiEndpoint,
		AuthProvider: authProvider,
		Version:      r.config.rorVersion,
		Role:         r.config.role,
	}
	transport := resttransport.NewRorHttpTransport(&clientConfig)
	r.rorAPIClient = rorclient.NewRorClient(transport)
	if err := r.rorAPIClient.Ping(); err != nil {
		return fmt.Errorf("failed to ping RorClient: %w", err)
	}

	return nil
}
