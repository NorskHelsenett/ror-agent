package healthservice

import (
	"github.com/NorskHelsenett/ror/pkg/config/configconsts"
	"github.com/NorskHelsenett/ror/pkg/config/rorconfig"
	healthserver "github.com/NorskHelsenett/ror/pkg/helpers/rorhealth/server"
	"github.com/NorskHelsenett/ror/pkg/rlog"
)

func MustStart() {
	rlog.Info("Initializing health server")
	err := healthserver.Start(healthserver.ServerString(rorconfig.GetString(configconsts.HEALTH_ENDPOINT)))

	if err != nil {
		rlog.Fatal("could not start health server", err)
	}
}
