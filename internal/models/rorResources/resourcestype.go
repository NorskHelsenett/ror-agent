package rorResources

import (
	"crypto/md5" // #nosec G501 - MD5 is used for hash calculation only
	"encoding/json"
	"fmt"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type rorResource struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Uid        string `json:"uid"`
	Hash       string `json:"hash"`
	Resource   any    `json:"resource"`
}

func NewFromUnstructured(input *unstructured.Unstructured) (rorResource, error) {
	if input == nil {
		return rorResource{}, fmt.Errorf("input is nil")
	}
	if input.Object == nil {
		return rorResource{}, fmt.Errorf("input.Object is nil")
	}

	returnResource := rorResource{
		ApiVersion: input.GetAPIVersion(),
		Kind:       input.GetKind(),
		Uid:        string(input.GetUID()),
	}

	removeUnnecessaryDataFromObject(input.Object)

	h, err := calculateHashFromObject(input.Object)
	if err != nil {
		return returnResource, err
	}
	returnResource.Hash = h
	if err := getResourceFromObject(&returnResource, input.Object); err != nil {
		return returnResource, err
	}
	return returnResource, nil
}

func prepareResourcePayloadFromObject[D any](obj map[string]any) (D, error) {
	var outStruct D
	if obj == nil {
		return outStruct, fmt.Errorf("obj is nil")
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj, &outStruct); err != nil {
		rlog.Error("error converting unstructured object", err)
		return outStruct, err
	}
	return outStruct, nil
}

func calculateHashFromObject(obj map[string]any) (string, error) {
	hashObj := make(map[string]any, len(obj))
	for k, v := range obj {
		hashObj[k] = v
	}

	md, ok := obj["metadata"].(map[string]any)
	if !ok || md == nil {
		md = map[string]any{}
	}
	mdCopy := make(map[string]any, len(md))
	for k, v := range md {
		mdCopy[k] = v
	}

	// Match previous behavior: set these fields to null before hashing.
	mdCopy["resourceVersion"] = nil
	mdCopy["creationTimestamp"] = nil
	mdCopy["generation"] = nil
	hashObj["metadata"] = mdCopy

	input, err := json.Marshal(hashObj)
	if err != nil {
		rlog.Error("error marshaling json for hash", err)
		return "", err
	}

	resourceHash := fmt.Sprintf("%x", md5.Sum(input)) // #nosec G401 - MD5 is used for hash calculation only
	return resourceHash, nil
}

func removeUnnecessaryDataFromObject(obj map[string]any) {
	md, ok := obj["metadata"].(map[string]any)
	if !ok || md == nil {
		return
	}
	ann, ok := md["annotations"].(map[string]any)
	if !ok || ann == nil {
		return
	}
	delete(ann, "kubectl.kubernetes.io/last-applied-configuration")
	if len(ann) == 0 {
		delete(md, "annotations")
	}
}

func (r rorResource) NewResourceUpdateModel(owner apiresourcecontracts.ResourceOwnerReference, action apiresourcecontracts.ResourceAction) *apiresourcecontracts.ResourceUpdateModel {
	return &apiresourcecontracts.ResourceUpdateModel{
		Owner:      owner,
		ApiVersion: r.ApiVersion,
		Kind:       r.Kind,
		Uid:        r.Uid,
		Action:     action,
		Hash:       r.Hash,
		Resource:   r.Resource,
	}
}
