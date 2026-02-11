// The package implements clients for the ror-agent
package clusteragentclient

import (
	"context"
	"fmt"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/apikeystypes/v2"
	kubernetesclient "github.com/NorskHelsenett/ror/pkg/clients/kubernetes"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpauthprovider"
	"github.com/NorskHelsenett/ror/pkg/clients/rorclient/transports/resttransport/httpclient"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"
	"github.com/NorskHelsenett/ror/pkg/helpers/idhelper"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/clusterinterregator/v2"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/interregatortypes/v2"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/providers/providermodels"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels"
	"github.com/NorskHelsenett/ror/pkg/models/aclmodels/rorresourceowner"
	identitymodels "github.com/NorskHelsenett/ror/pkg/models/identity"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	UNKNOWN_CLUSTER_ID = "___unknown_cluster_id___"
	UNKNOWN_API_KEY    = "___unknown_api_key___"
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
	rorconfig.SetDefault(configconsts.API_KEY, UNKNOWN_API_KEY)
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

	err = client.initRorAgentClientSetup()
	if err != nil {
		rlog.Error("failed to initialize kubernetes cluster setup", err)
		return nil, err
	}

	// err = client.initAuthorizedRorClient()
	// if err != nil {
	// 	rlog.Error("failed to setup RorClient", err)
	// 	return nil, err
	// }

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

// initRorAgentClientSetup initializes the kubernetes cluster setup by verifying access to the namespace, api endpoint and cluster id.
func (r *rorAgentClient) initRorAgentClientSetup() error {

	// check if namespace is accessible
	_, err := r.k8sClientSet.GetNamespace(r.config.namespace)
	if err != nil {
		return fmt.Errorf("failed to get namespace %s", err)
	}

	err = r.getClusterAuthFromSecret()
	if err != nil {
		return fmt.Errorf("Could not get cluster auth from secret: %s", err)
	}

	// check if api endpoint is accessible

	// err = client.initAuthorizedRorClient()
	// if err != nil {
	// 	rlog.Error("failed to setup RorClient", err)
	// 	return nil, err
	// }

	// ver, err := client.rorAPIClient.Info().GetVersion(context.TODO())
	// if err != nil {
	// 	return nil, err
	// }

	// selfdata, err := client.rorAPIClient.Clusters().GetSelf()
	// if err != nil {
	// 	return nil, err
	// }

	//Check if we know the cluster id

	interregatorClusterid := r.config.interregator.GetClusterId()

	// ClusterID unknown in both secret and interregator
	// Will failover to asking the api for existing clusterid if apikey is set
	if (r.config.clusterId == UNKNOWN_CLUSTER_ID) && (interregatorClusterid == providermodels.UNKNOWN_CLUSTER_ID) && (r.config.apiKey != UNKNOWN_API_KEY) {
		rlog.Info("Trying to ask the api for existing cluster id")
		err = r.initAuthorizedRorClient()
		if err != nil {
			rlog.Error("failed to setup RorClient", err)
			return err
		}
		selfdata, err := r.rorAPIClient.V2().Self().Get()
		if err != nil {
			return err
		}

		if selfdata.Type != identitymodels.IdentityTypeCluster {
			err = fmt.Errorf("wrong type of apikey in secret")
			return err
		}

		r.config.clusterId = selfdata.User.Name
		err = r.kubernetesUpdateOrCreateApiKeySecret()
		if err != nil {
			return fmt.Errorf("failed to update api key secret with cluster id %s", err)
		}
		rlog.Info("Using cluster id from api", rlog.String("cluster id", r.config.clusterId))
	}

	// Warn if cluster id in secret does not match interregator cluster id
	// If both are known
	if r.config.clusterId != interregatorClusterid && interregatorClusterid != providermodels.UNKNOWN_CLUSTER_ID {
		rlog.Warn("cluster id in secret does not match interregator cluster id, using secret cluster id",
			rlog.String("secret cluster id", r.config.clusterId),
			rlog.String("interregator cluster id", interregatorClusterid))
	}

	// Use interregator cluster id if secret cluster id is unknown and interregator cluster id is known
	if r.config.clusterId == UNKNOWN_CLUSTER_ID && interregatorClusterid != providermodels.UNKNOWN_CLUSTER_ID {
		rlog.Info("Using cluster id from interregator", rlog.String("cluster id", interregatorClusterid))
		r.config.clusterId = interregatorClusterid
		err = r.kubernetesUpdateOrCreateApiKeySecret()
		if err != nil {
			return fmt.Errorf("failed to update api key secret with cluster id %s", err)
		}
	}

	// If no cluster id is found, generate new cluster id
	if r.config.clusterId == UNKNOWN_CLUSTER_ID {
		rlog.Info("cluster id not found in secret or interregator, generating new cluster id")
		clustername := r.config.interregator.GetClusterName()
		if clustername == "" {
			err = fmt.Errorf("Could not get clustername, failing")
			return err
		}
		r.config.clusterId = idhelper.GetIdentifier(clustername)
		err = r.kubernetesUpdateOrCreateApiKeySecret()
		if err != nil {
			return fmt.Errorf("failed to update api key secret with cluster id %s", err)
		}
	}

	r.config.apiKey = UNKNOWN_API_KEY
	// TODO: use v2 og clusters/register endpoint to register cluster if cluster id is unknown
	if r.config.apiKey == UNKNOWN_API_KEY {
		rlog.Info("api key secret not found, registering new key")

		r.initUnathorizedRorClient()
		resp, err := r.rorAPIClient.ApiKeysV2().RegisterAgent(apikeystypes.RegisterClusterRequest{
			ClusterId: r.config.clusterId,
		})
		if err != nil {
			return fmt.Errorf("failed to register cluster %s", err)
		}
		if resp.ApiKey != resp.ClusterId {
			rlog.Info("The api changed the cluster id during registration", rlog.String("old cluster id", r.config.clusterId), rlog.String("new cluster id", resp.ClusterId))
		}

		r.config.clusterId = resp.ClusterId
		r.config.apiKey = resp.ApiKey
		// Create or update the secret with new api key and cluster id
		err = r.kubernetesUpdateOrCreateApiKeySecret()
		if err != nil {
			return fmt.Errorf("failed to create api key secret %s", err)
		}

	}

	// Setting the config env values for cluster id and api key for backward compatibility
	rorconfig.Set(configconsts.CLUSTER_ID, r.config.clusterId)
	rorconfig.Set(configconsts.API_KEY, r.config.apiKey)
	return nil
}

func (r *rorAgentClient) kubernetesCreateApiKeySecret() error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.config.apiKeySecret,
			Namespace: r.config.namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"APIKEY":     r.config.apiKey,
			"CLUSTER_ID": r.config.clusterId,
		},
	}
	_, err := r.k8sClientSet.CreateSecret(r.config.namespace, secret)
	if err != nil {
		rlog.Error("failed to create api key secret", err)
		return err
	}
	return nil

}
func (r *rorAgentClient) kubernetesUpdateOrCreateApiKeySecret() error {
	var hasChanged bool
	secret, err := r.k8sClientSet.GetSecret(r.config.namespace, r.config.apiKeySecret)
	// ensure secret exists before attempting to update it

	if err != nil {
		if errors.IsNotFound(err) {
			rlog.Info("api key secret missing during update, creating new secret")
			return r.kubernetesCreateApiKeySecret()
		}
		rlog.Error("failed to get api key secret for update", err)
		return err
	}

	if r.config.apiKey != UNKNOWN_API_KEY && string(secret.Data["APIKEY"]) != r.config.apiKey {
		secret.Data["APIKEY"] = []byte(r.config.apiKey)
		hasChanged = true
	}
	if r.config.clusterId != UNKNOWN_CLUSTER_ID && string(secret.Data["CLUSTER_ID"]) != r.config.clusterId {
		secret.Data["CLUSTER_ID"] = []byte(r.config.clusterId)
		hasChanged = true
	}
	if !hasChanged {
		return nil
	}
	_, err = r.k8sClientSet.SetSecret(r.config.namespace, secret)
	if err != nil {
		rlog.Error("failed to update api key secret", err)
		return err
	}
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

	if r.config.apiKey == UNKNOWN_API_KEY {
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
	if err := r.rorAPIClient.CheckConnection(); err != nil {
		return fmt.Errorf("failed to ping RorClient: %w", err)
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
