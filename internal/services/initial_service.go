package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/NorskHelsenett/ror-agent/internal/clients/clients"
	"github.com/NorskHelsenett/ror-agent/internal/kubernetes/operator/initialize"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/clusterinterregator/v2"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

var EgressIp string

func CreateApikey(k8sClient *kubernetes.Clientset, metricsClient *metrics.Clientset) (string, error) {
	//clusterInfo, err := initialize.GetClusterInfoFromNode(k8sClient, metricsClient)
	interregator := clusterinterregator.NewClusterInterregatorFromKubernetesClient(k8sClient)

	if interregator == nil {
		err := errors.New("failed to get apikey, could not create cluster interregator")
		return "", err
	}

	rorUrl := rorconfig.GetString(configconsts.API_ENDPOINT)
	apikey, err := initialize.GetApikey(interregator, rorUrl)
	if err != nil {
		rlog.Error("not able to get api key", err,
			rlog.String("clusterName", interregator.GetClusterName()),
			rlog.String("ror url", rorUrl))

		return "", fmt.Errorf("could not get api key from API (url: %s)", rorUrl)
	}
	rorconfig.Set(configconsts.API_KEY, apikey)
	return apikey, nil
}

func ExtractApikeyOrDie() error {
	k8sClient, err := clients.Kubernetes.GetKubernetesClientset()
	if err != nil {
		return err
	}

	metricsClient, err := clients.Kubernetes.GetMetricsClient()
	if err != nil {
		return err
	}

	secretName := rorconfig.GetString(configconsts.API_KEY_SECRET)
	namespace := rorconfig.GetString(configconsts.POD_NAMESPACE)
	secretApiKey := "APIKEY"
	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metaV1.GetOptions{})
	if err != nil {
		apikey, err := CreateApikey(k8sClient, metricsClient)
		if err != nil {
			rlog.Error("could not create api key: ", err)
			return errors.New("could not create api key")
		}
		secret, err = k8sClient.CoreV1().Secrets(namespace).Create(context.TODO(),
			&coreV1.Secret{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Type: "Opaque",
				StringData: map[string]string{
					secretApiKey: apikey,
				},
			},
			metaV1.CreateOptions{})
		if err != nil {
			rlog.Error("could not create k8s secret: ", err)
			return errors.New("could not create secret")
		}
	}

	apikey := string(secret.Data[secretApiKey])
	rorconfig.Set(configconsts.API_KEY, apikey)

	return nil
}

func GetEgressIp() {
	internettCheck := "https://api.ipify.org/"
	nhnCheck := "ip.nhn.no"
	_, err := net.LookupIP(nhnCheck)
	var apiHost string
	if err != nil {
		apiHost = internettCheck
	} else {
		apiHost = fmt.Sprintf("http://%s", nhnCheck)
	}

	rlog.Info("Resolving ip", rlog.String("api host", apiHost))
	res, err := http.Get(apiHost) // #nosec G107 - we are not using user input
	if err != nil {
		// assuming retry but on internett
		apiHost = internettCheck
		res, err = http.Get(apiHost) // #nosec G107 - we are not using user input
		if err != nil {
			errorMsg := fmt.Sprintf("could not reach host %s", apiHost)
			rlog.Info(errorMsg)
			return
		}
	}

	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode > 299 {
		rlog.Info("response failed", rlog.Int("status code", res.StatusCode), rlog.ByteString("body", body))
		return
	}

	if err != nil {
		rlog.Error("could not parse body", err)
		return
	}

	EgressIp = strings.Replace(string(body), "\n", "", -1)
}
