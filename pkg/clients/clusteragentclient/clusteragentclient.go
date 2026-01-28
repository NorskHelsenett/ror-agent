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

const (
	ERR_SECRET_NOT_FOUND = "___secret_not_found___"
	UNKNOWN_CLUSTER_ID   = "unknown_cluster_id"
	UNKNOWN_API_KEY      = "unknown_api_key"
)

type RorAgentClientInterface interface {
	GetRorClient() rorclient.RorClientInterface
	GetKubernetesClientset() *kubernetesclient.K8sClientsets

	PingRorAPI() error
}

type RorAgentClientConfig struct {
	role         string
	namespace    string
	apiEndpoint  string
	clusterId    string
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

func NewRorAgentClientWithDefaults() (RorAgentClientInterface, error) {
	return NewRorAgentClient(GetDefaultRorAgentClientConfig())
}

func NewRorAgentClient(config *RorAgentClientConfig) (RorAgentClientInterface, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil, please use NewRorAgentClientWithDefaults if no custom config is needed")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	client := &rorAgentClient{
		config:       *config,
		k8sClientSet: kubernetesclient.MustInitializeKubernetesClient(),
	}

	err := client.initInterregator()
	if err != nil {
		rlog.Error("failed to initialize interregator", err)
		return nil, err
	}

	err = client.getClusterAuthFromSecret()
	if err != nil {
		return nil, fmt.Errorf("Could not set cluster auth from secret: %s", err)
	}

	err = client.initKubernetesClusterSetup()
	if err != nil {
		rlog.Error("failed to initialize kubernetes cluster setup", err)
		return nil, err
	}

	err = client.initAuthorizedRorClient()
	if err != nil {
		rlog.Error("failed to setup RorClient", err)
		return nil, err
	}

	ver, err := client.rorAPIClient.Info().GetVersion(context.TODO())
	if err != nil {
		return nil, err
	}

	selfdata, err := client.rorAPIClient.Clusters().GetSelf()
	if err != nil {
		return nil, err
	}

	rlog.Info("connected to ror-api", rlog.String("version", ver), rlog.String("clusterid", selfdata.ClusterId))
	client.rorAPIClient.SetOwnerref(rorresourceowner.RorResourceOwnerReference{
		Scope:   aclmodels.Acl2ScopeCluster,
		Subject: aclmodels.Acl2Subject(selfdata.ClusterId),
	})
	rorconfig.Set(configconsts.CLUSTER_ID, selfdata.ClusterId)

	return client, nil
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

// getClusterAuthFromSecret sets the clusterI and api-key from the secret defined in the config value apiKeySecret
// if the secret is not found, it will return an error.
// if clusterId is not found in the secret, it will set it to UNKNOWN_CLUSTER_ID
func (r *rorAgentClient) getClusterAuthFromSecret() error {
	rlog.Debug("Using kubernetes secret to get api-key")
	secret, err := r.k8sClientSet.GetSecret(r.config.namespace, r.config.apiKeySecret)
	if err != nil {
		if errors.IsNotFound(err) {
			rlog.Warn("api key secret not found")
			r.config.clusterId = UNKNOWN_CLUSTER_ID
			r.config.apiKey = UNKNOWN_API_KEY
			return nil
		} else {
			rlog.Error("failed to get api key secret", err)
			return err
		}
	}

	r.config.clusterId = string(secret.Data["CLUSTER_ID"])
	if r.config.clusterId == "" {
		r.config.clusterId = UNKNOWN_CLUSTER_ID
	}
	r.config.apiKey = string(secret.Data["APIKEY"])
	if r.config.apiKey == "" {
		r.config.apiKey = UNKNOWN_API_KEY
	}
	return nil
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
	if !r.rorAPIClient.Ping() {
		return fmt.Errorf("failed to ping RorClient")
	}

	return nil
}

func (c *rorAgentClient) initInterregator() error {
	k8sclientset, err := c.k8sClientSet.GetKubernetesClientset()
	if err != nil {
		rlog.Error("failed to get kubernetes clientset", err)
		return err
	}

	c.config.interregator = clusterinterregator.NewClusterInterregatorFromKubernetesClient(k8sclientset)

	// Verify that we know the provider type
	if c.config.interregator.GetProvider() == providermodels.ProviderTypeUnknown {
		rlog.Error("could not determine provider type", fmt.Errorf("unknown provider"))
		return fmt.Errorf("could not determine provider type")
	}
	return nil
}

func (c *RorAgentClientConfig) Validate() error {
	if c.role == "" {
		return fmt.Errorf("role cannot be empty")
	}
	if c.namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if c.apiEndpoint == "" {
		return fmt.Errorf("apiEndpoint cannot be empty")
	}
	if c.apiKeySecret == "" {
		return fmt.Errorf("apiKeySecret cannot be empty")
	}
	return nil

}
