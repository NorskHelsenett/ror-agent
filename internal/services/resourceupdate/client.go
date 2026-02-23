package resourceupdate

import (
	"github.com/NorskHelsenett/ror-agent/internal/clients/clients"
	"github.com/NorskHelsenett/ror-agent/internal/config"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/go-resty/resty/v2"
)

// the function sends the resource to the ror api. If receiving a non 2xx statuscode it will retun an error.
func sendResourceUpdateToRor(resourceUpdate *apiresourcecontracts.ResourceUpdateModel) error {
	rorClient, err := clients.GetOrCreateRorClient()
	if err != nil {
		rlog.Error("Could not get ror-api client", err)
		config.IncreaseErrorCount()
		return err
	}
	var url string
	var response *resty.Response

	switch resourceUpdate.Action {
	case apiresourcecontracts.K8sActionAdd:
		url = "/v1/resources"
		response, err = rorClient.R().
			SetHeader("Content-Type", "application/json").
			SetBody(resourceUpdate).
			Post(url)

	case apiresourcecontracts.K8sActionUpdate:
		url = "/v1/resources/uid/" + resourceUpdate.Uid
		response, err = rorClient.R().
			SetHeader("Content-Type", "application/json").
			SetBody(resourceUpdate).
			Put(url)
	case apiresourcecontracts.K8sActionDelete:
		url = "/v1/resources/uid/" + resourceUpdate.Uid
		response, err = rorClient.R().
			SetHeader("Content-Type", "application/json").
			SetBody(resourceUpdate).
			Delete(url)
	}

	if err != nil {
		config.IncreaseErrorCount()
		rlog.Error("could not send data to ror-api", err,
			rlog.Int("error count", config.ErrorCount))
		return err
	}

	if response == nil {
		config.IncreaseErrorCount()
		rlog.Error("response is nil", err,
			rlog.Int("error count", config.ErrorCount))
		return err
	}

	if !response.IsSuccess() {
		config.IncreaseErrorCount()
		rlog.Info("got non 200 statuscode from ror-api", rlog.Int("status code", response.StatusCode()),
			rlog.Int("error count", config.ErrorCount))
		return err
	} else {
		config.ResetErrorCount()
		rlog.Debug("partial update sent to ror", rlog.String("api verson", resourceUpdate.ApiVersion), rlog.String("kind", resourceUpdate.Kind), rlog.String("uid", resourceUpdate.Uid))
	}
	return nil
}
