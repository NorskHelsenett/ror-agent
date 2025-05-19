package config

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
	version    = rorversion.NewRorVersion(VersionDefault, CommitDefault)
	ErrorCount int
)

func Init() {
	rlog.InitializeRlog()
	rlog.Info("Configuration initializing ...")
	viper.SetDefault(configconsts.VERSION, VersionDefault)
	viper.SetDefault(configconsts.COMMIT, CommitDefault)
	viper.SetDefault(configconsts.HEALTH_ENDPOINT, ":8100")
	viper.SetDefault(configconsts.POD_NAMESPACE, "ror")
	viper.SetDefault(configconsts.API_KEY_SECRET, "ror-apikey")
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
