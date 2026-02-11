// The package implements clients for the ror-agent
package clients

import (
	"fmt"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/config/rorversion"

	"github.com/go-resty/resty/v2"
)

var client *resty.Client

// DEPRECATED: use GetRorClient from pkg/clients/rorclient/client.go
func GetOrCreateRorClient() (*resty.Client, error) {
	if client != nil {
		return client, nil
	}

	client = resty.New()
	client.SetBaseURL(rorconfig.GetString(configconsts.API_ENDPOINT))
	client.Header.Add("X-API-KEY", rorconfig.GetString(configconsts.API_KEY))
	client.Header.Set("User-Agent", fmt.Sprintf("ROR-Agent/%s", rorversion.GetRorVersion().GetVersion()))

	return client, nil
}
