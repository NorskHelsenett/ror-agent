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
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"

	"github.com/NorskHelsenett/ror/pkg/config/rorversion"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/spf13/viper"
)

const (
	VersionDefault = "0.1.2"
	CommitDefault  = "FFFFF"
)

var (
	version = rorversion.NewRorVersion(VersionDefault, CommitDefault)
)

var (
	ErrorCount int
)

func Init() {
	rlog.InitializeRlog()
	rlog.Info("Configuration initializing ...")
	viper.SetDefault(configconsts.VERSION, VersionDefault)
	viper.SetDefault(configconsts.ROLE, "ClusterAgent")
	viper.SetDefault(configconsts.COMMIT, CommitDefault)
	viper.SetDefault(configconsts.POD_NAMESPACE, "ror")
	viper.SetDefault(configconsts.API_KEY_SECRET, "ror-apikey")
	viper.SetDefault(configconsts.HEALTH_ENDPOINT, ":9998")
	viper.AutomaticEnv()
	version = rorversion.NewRorVersion(viper.GetString(configconsts.VERSION), viper.GetString(configconsts.COMMIT))
	viper.Set(configconsts.VERSION, version.Version)
	viper.Set(configconsts.COMMIT, version.Commit)
}

func IncreaseErrorCount() {
	ErrorCount++
}

func ResetErrorCount() {
	ErrorCount = 0
}

func GetRorVersion() rorversion.RorVersion {
	return version
}
