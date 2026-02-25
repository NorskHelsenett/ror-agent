package pprofservice

import (
	"net/http"

	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	"github.com/NorskHelsenett/ror/pkg/rlog"
)

func MayStartPprof() {
	if rorconfig.GetBool(configconsts.ENABLE_PPROF) {
		go func() {
			rlog.Info("Starting pprof server on port 6060")
			err := http.ListenAndServe("localhost:6060", nil)
			if err != nil {
				rlog.Error("could not start pprof server", err)
			}
		}()
	}
}
