//This package contains the variables used by the agentconfig package
//It also contains the Init function which initializes the configuration
//and sets the default values for the configuration variables
//It also contains the IncreaseErrorCount and ResetErrorCount functions
//which are used to increase the error count and reset the error count respectively
//It also contains the GetRorVersion function which returns the RorVersion struct
//which contains the version and commit of the agent
//
// The version and commit are set at compile time using the ldflags
// The version and commit are set to the default  v1.1.0/FFFFF if not set at compile time
//
// The configuration variables are set to the default values if not set in the environment
// The environment variables required are:
// ROLE default value is ClusterAgent
// HEALTH_ENDPOINT default value is :8100
// POD_NAMESPACE default value is ror
// API_KEY_SECRET default value is ror-apikey
// API_KEY migt be provided by the user if a secret containing the api key is not present in the cluster
//

package agentconfig

import (
	"github.com/NorskHelsenett/ror-agent/pkg/config/agentconsts"
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"

	"github.com/NorskHelsenett/ror/pkg/rlog"
)

var (
	ErrorCount int
)

func Init() {
	rlog.InitializeRlog()
	rlog.Info("Configuration initializing ...")
	rorconfig.SetDefault(configconsts.ROLE, "ClusterAgent")
	rorconfig.SetDefault(configconsts.POD_NAMESPACE, "ror")
	rorconfig.SetDefault(configconsts.API_KEY_SECRET, "ror-apikey")
	rorconfig.SetDefault(configconsts.HEALTH_ENDPOINT, ":9998")

	rorconfig.SetDefault(configconsts.ENABLE_PPROF, false)
	rorconfig.SetDefault(agentconsts.DynamicWatchNoCacheEnv, true)
	rorconfig.SetDefault(agentconsts.ForceGCAfterInitialListEnv, true)

	rorconfig.AutomaticEnv()

}

func IncreaseErrorCount() {
	ErrorCount++
}

func ResetErrorCount() {
	ErrorCount = 0
}
