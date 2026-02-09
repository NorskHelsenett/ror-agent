// authservice implements authorization helpers for the agent
package authservice

import (
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"

	"github.com/NorskHelsenett/ror/pkg/models/aclmodels"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"
)

// creaters a ownerref object for the agent
func CreateOwnerref() apiresourcecontracts.ResourceOwnerReference {
	return apiresourcecontracts.ResourceOwnerReference{
		Scope:   aclmodels.Acl2ScopeCluster,
		Subject: rorconfig.GetString(configconsts.CLUSTER_ID),
	}
}
