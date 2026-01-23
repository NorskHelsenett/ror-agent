package config

import (
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
	rorconfig.SetDefault(configconsts.HEALTH_ENDPOINT, ":8100")
	rorconfig.SetDefault(configconsts.POD_NAMESPACE, "ror")
	rorconfig.SetDefault(configconsts.API_KEY_SECRET, "ror-apikey")
	rorconfig.AutomaticEnv()
}

func IncreaseErrorCount() {
	ErrorCount++
}
func ResetErrorCount() {
	ErrorCount = 0
}
