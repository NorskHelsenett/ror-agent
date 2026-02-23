package config

import (
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"

	"github.com/NorskHelsenett/ror/pkg/rlog"
)

const (
	DynamicWatchNoCacheEnv                 = "ROR_DYNAMIC_WATCH_NO_CACHE"
	ForceGCAfterInitialListEnv             = "ROR_FORCE_GC_AFTER_INITIAL_LIST"
	ForceGCAfterInitialListFreeOSMemoryEnv = "ROR_FORCE_GC_AFTER_INITIAL_LIST_FREE_OS_MEMORY"
)

var (
	ErrorCount int
)

func Init() {
	rlog.InitializeRlog()
	rlog.Info("Configuration initializing ...")
	rorconfig.InitConfig()
	rorconfig.SetDefault(configconsts.HEALTH_ENDPOINT, ":8100")
	rorconfig.SetDefault(configconsts.POD_NAMESPACE, "ror")
	rorconfig.SetDefault(configconsts.API_KEY_SECRET, "ror-apikey")
	rorconfig.SetDefault(configconsts.ENABLE_PPROF, false)
	rorconfig.SetDefault(DynamicWatchNoCacheEnv, true)
	rorconfig.SetDefault(ForceGCAfterInitialListEnv, true)
	rorconfig.SetDefault(configconsts.ROLE, "ror-agent")

	rorconfig.AutomaticEnv()
}

func IncreaseErrorCount() {
	ErrorCount++
}
func ResetErrorCount() {
	ErrorCount = 0
}
