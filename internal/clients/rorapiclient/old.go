// TODO: This internal package is copied from ror, should determine if its common and should be moved to ror/pkg
package rorapiclient

import (
	"fmt"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
)

var rorclientnonauth *resty.Client

// Deprecated: GetOrCreateRorRestyClientNonAuth is deprecated. Use rorclient instead
func GetOrCreateRorRestyClientNonAuth() (*resty.Client, error) {
	if rorclientnonauth != nil {
		return rorclientnonauth, nil
	}

	rorclientnonauth = resty.New()
	rorclientnonauth.SetBaseURL(viper.GetString(configconsts.API_ENDPOINT))
	rorclientnonauth.Header.Set("User-Agent", fmt.Sprintf("ROR-Agent/%s", viper.GetString(configconsts.VERSION)))
	return rorclientnonauth, nil
}
