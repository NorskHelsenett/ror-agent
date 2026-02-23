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
	rorconfig.SetDefault(configconsts.ENABLE_PPROF, false)
	rorconfig.SetDefault("ROR_DYNAMIC_WATCH_NO_CACHE", true)
	rorconfig.SetDefault("ROR_FORCE_GC_AFTER_INITIAL_LIST", true)
	rorconfig.SetDefault("ROR_FORCE_GC_AFTER_INITIAL_LIST_MIN_INTERVAL_SECONDS", 2)
	rorconfig.AutomaticEnv()
}

func IncreaseErrorCount() {
	ErrorCount++
}
func ResetErrorCount() {
	ErrorCount = 0
}
