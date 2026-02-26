package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/NorskHelsenett/ror-agent/internal/kubernetes/k8smodels"
	"github.com/NorskHelsenett/ror-agent/internal/models/argomodels"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/rlog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func getNhnToolingMetadata(rorClientInterface clusteragentclient.RorAgentClientInterface) (k8smodels.NhnTooling, error) {
	result := k8smodels.NhnTooling{
		Version:      MissingConst,
		Branch:       MissingConst,
		AccessGroups: []string{},
		Environment:  "dev",
	}

	k8sClient, err := rorClientInterface.GetKubernetesClientset().GetKubernetesClientset()
	if err != nil {
		return result, err
	}

	dynamicClient, err := rorClientInterface.GetKubernetesClientset().GetDynamicClient()
	if err != nil {
		return result, err
	}

	namespace := rorconfig.GetString(configconsts.POD_NAMESPACE)
	nhnToolingConfigMap, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "nhn-tooling", v1.GetOptions{
		TypeMeta:        v1.TypeMeta{},
		ResourceVersion: "",
	})

	if err != nil {
		return result, fmt.Errorf("could not find config map %s for ror in namespace %s", "nhn-tooling", namespace)
	}

	if nhnToolingConfigMap.Data == nil {
		return result, errors.New("no data in config map for ror")
	}

	environment := nhnToolingConfigMap.Data["environment"]
	toolingVersion := nhnToolingConfigMap.Data["toolingVersion"]

	accessGroups := NewAccessGroupsFromData(nhnToolingConfigMap.Data)

	if environment == "" {
		environment = "dev"
	}

	if toolingVersion == "" {
		toolingVersion = MissingConst
	}

	branch := MissingConst
	nhnToolingApp, err := getNhnToolingInfo(dynamicClient)
	if err != nil {
		rlog.Error("could not get nhn-tooling application", err)
	} else {
		branch = nhnToolingApp.Spec.Source.TargetRevision
		if len(nhnToolingApp.Status.Sync.Revision) < 20 {
			toolingVersion = nhnToolingApp.Status.Sync.Revision
		}
	}

	result.Version = toolingVersion
	result.Environment = environment
	result.AccessGroups = accessGroups.StringArray()
	result.Branch = branch

	return result, nil
}

func getNhnToolingInfo(dynamicClient dynamic.Interface) (argomodels.Application, error) {
	result := argomodels.Application{}
	applications, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}).
		Namespace("argocd").
		Get(context.TODO(), "nhn-tooling", v1.GetOptions{})
	if err != nil {
		rlog.Error("could not get nhn-tooling application", err)
		return result, err
	}

	appByteArray, err := applications.MarshalJSON()
	if err != nil {
		rlog.Error("could not marshal application", err)
		return result, err
	}

	var nhnTooling argomodels.Application
	err = json.Unmarshal(appByteArray, &nhnTooling)
	if err != nil {
		rlog.Error("could not marshal applications", err)
		return result, err
	}

	appByteArray = nil // Clear the byte array to free up memory

	if nhnTooling.Metadata.Name == "" {
		return result, errors.New("could not find nhn-tooling application")
	}

	return nhnTooling, nil
}
