package resourceupdatev2

import (
	"context"
	"fmt"

	"github.com/NorskHelsenett/ror-agent/v2/internal/clients"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/v2/apicontractsv2resources"
)

func InitHashList() (*apicontractsv2resources.HashList, error) {
	rorclient := clients.RorConfig.GetRorClient()
	hashList, err := rorclient.ResourceV2().GetOwnHashes(context.TODO(), clients.RorConfig.GetClusterId())
	if err != nil {
		fmt.Println("Error getting hashlist from api", err)
		return nil, err
	}
	return hashList, nil

}
