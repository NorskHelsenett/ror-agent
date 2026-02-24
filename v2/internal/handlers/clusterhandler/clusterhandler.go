package clusterhandler

import (
	"github.com/NorskHelsenett/ror/pkg/rlog"
)

func MustStart() {
	err := Start()
	if err != nil {
		rlog.Fatal("could not start cluster handler", err)
	}
}

func Start() error {
	rlog.Info("Starting cluster handler")
	return nil
}
