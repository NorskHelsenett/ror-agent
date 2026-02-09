// The package implements clients for the ror-agent
package clusteragentclient

import (
	"context"
	"fmt"

	"github.com/NorskHelsenett/ror/pkg/apicontracts"
	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpauthprovider"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpclient"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/clusterinterregator/v2"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/interregatortypes/v2"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/providers/providermodels"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels/rorresourceowner"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

const ERR_SECRET_NOT_FOUND = "___secret_not_found___"

type RorAgentClientInterface interface {
	GetRorClient() rorclient.RorClientInterface
	GetKubernetesClientset() *kubernetesclient.K8sClientsets

	PingRorAPI() error
}

type RorAgentClientConfig struct {
	role         string
	namespace    string
	apiEndpoint  string
	apiKey       string
	apiKeySecret string
	interregator interregatortypes.ClusterInterregator
}

type rorAgentClient struct {
	rorAPIClient *rorclient.RorClient
	k8sClientSet *kubernetesclient.K8sClientsets
	config       RorAgentClientConfig
}

func GetDefaultRorAgentClientConfig() *RorAgentClientConfig {
	rorconfig.SetDefault(configconsts.API_KEY, ERR_SECRET_NOT_FOUND)
	return &RorAgentClientConfig{
		role:         rorconfig.GetString(configconsts.ROLE),
		namespace:    rorconfig.GetString(configconsts.POD_NAMESPACE),
		apiKeySecret: rorconfig.GetString(configconsts.API_KEY_SECRET),
		apiKey:       rorconfig.GetString(configconsts.API_KEY),
		apiEndpoint:  rorconfig.GetString(configconsts.API_ENDPOINT),
	}
}

func NewRorAgentClient(config *RorAgentClientConfig) (RorAgentClientInterface, error) {
	var err error

	// Create a pointer to the struct
	rorClient := &rorAgentClient{}

	if config != nil {
		rorClient.config = *config
	} else {
		rorClient.config = *GetDefaultRorAgentClientConfig()
	}

	rorClient.k8sClientSet = kubernetesclient.MustInitializeKubernetesClient()

	if rorClient.k8sClientSet != nil {
		k8sclientset, err := rorClient.k8sClientSet.GetKubernetesClientset()
		if err != nil {
			rlog.Error("failed to get kubernetes clientset", err)
			return nil, err
		}

		rorClient.config.interregator = clusterinterregator.NewClusterInterregatorFromKubernetesClient(k8sclientset)

		if rorClient.config.interregator.GetProvider() == providermodels.ProviderTypeUnknown {
			rlog.Error("could not determine provider type", fmt.Errorf("unknown provider"))
			return nil, fmt.Errorf("could not determine provider type")
		}

		rlog.Debug("Using kubernetes secret to get api-key")
		rorClient.kubernetesAuth()

		err = rorClient.initKubernetesClusterSetup()
		if err != nil {
			rlog.Error("failed to initialize kubernetes cluster setup", err)
			return nil, err
		}
	}

	err = rorClient.initAuthorizedRorClient()
	if err != nil {
		rlog.Error("failed to setup RorClient", err)
		return nil, err
	}

	ver, err := rorClient.rorAPIClient.Info().GetVersion(context.TODO())
	if err != nil {
		return nil, err
	}

	selfdata, err := rorClient.rorAPIClient.Clusters().GetSelf()
	if err != nil {
		return nil, err
	}

	rlog.Info("connected to ror-api", rlog.String("version", ver), rlog.String("clusterid", selfdata.ClusterId))
	rorClient.rorAPIClient.SetOwnerref(rorresourceowner.RorResourceOwnerReference{
		Scope:   aclmodels.Acl2ScopeCluster,
		Subject: aclmodels.Acl2Subject(selfdata.ClusterId),
	})
	rorconfig.Set(configconsts.CLUSTER_ID, selfdata.ClusterId)

	return rorClient, nil
}

func (r *rorAgentClient) GetRorClient() rorclient.RorClientInterface {
	return r.rorAPIClient
}

func (r *rorAgentClient) GetKubernetesClientset() *kubernetesclient.K8sClientsets {
	return r.k8sClientSet
}

func (r *rorAgentClient) PingRorAPI() error {
	if r.rorAPIClient == nil {
		r.initUnathorizedRorClient()
	}
	if r.rorAPIClient.Ping() {
		return nil
	}
	return fmt.Errorf("could not ping ror-api")
}

func (r *rorAgentClient) initKubernetesClusterSetup() error {

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

	//Check if we know the cluster id
	if r.config.interregator.GetClusterId() != providermodels.UNKNOWN_CLUSTER_ID {
		rlog.Info("cluster id found from interregator", rlog.String("cluster id", r.config.interregator.GetClusterId()))
	}

	if r.config.apiKey == ERR_SECRET_NOT_FOUND {
		rlog.Info("api key secret not found, interregating cluster and registering new key")

		r.initUnathorizedRorClient()
		key, err := r.rorAPIClient.Clusters().Register(apicontracts.AgentApiKeyModel{
			Identifier:     r.config.interregator.GetClusterName(),
			DatacenterName: r.config.interregator.GetDatacenter(),
			WorkspaceName:  r.config.interregator.GetClusterWorkspace(),
			Provider:       r.config.interregator.GetProvider(),
			Type:           apicontracts.ApiKeyTypeCluster,
		})
		if err != nil {
			return fmt.Errorf("failed to register cluster %s", err)
		}
		err = r.kubernetesCreateApiKeySecret(key)
		if err != nil {
			return fmt.Errorf("failed to create api key secret %s", err)
		}

	}
	rorconfig.Set(configconsts.API_KEY, r.config.apiKey)
	return nil
}

func (r *rorAgentClient) kubernetesCreateApiKeySecret(apiKey string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.config.apiKeySecret,
			Namespace: r.config.namespace,
		},
		Type: corev1.SecretTypeOpaque,
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

// kubernetesAuth sets the api key from the secret defined in the config value apiKeySecret
// in the namespace defined in the config value namespace
// if the secret does not exist, it will be created
func (r *rorAgentClient) kubernetesAuth() {

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

func (r *rorAgentClient) initUnathorizedRorClient() {
	httptransportconfig := httpclient.HttpTransportClientConfig{
		BaseURL:      r.config.apiEndpoint,
		AuthProvider: httpauthprovider.NewNoAuthprovider(),
		Role:         r.config.role,
		Version:      rorversion.GetRorVersion(),
	}
	rorclienttransport := resttransport.NewRorHttpTransport(&httptransportconfig)
	r.rorAPIClient = rorclient.NewRorClient(rorclienttransport)
}
func (r *rorAgentClient) initAuthorizedRorClient() error {

	if r.config.apiKey == ERR_SECRET_NOT_FOUND {
		return fmt.Errorf("API_KEY is not set in the configuration")
	}
	authProvider := httpauthprovider.NewAuthProvider(httpauthprovider.AuthPoviderTypeAPIKey, r.config.apiKey)
	clientConfig := httpclient.HttpTransportClientConfig{
		BaseURL:      r.config.apiEndpoint,
		AuthProvider: authProvider,
		Version:      rorversion.GetRorVersion(),
		Role:         r.config.role,
	}
	transport := resttransport.NewRorHttpTransport(&clientConfig)
	r.rorAPIClient = rorclient.NewRorClient(transport)
	if err := r.rorAPIClient.Ping(); err != nil {
		return fmt.Errorf("failed to ping RorClient: %w", err)
	}

	return nil
}
