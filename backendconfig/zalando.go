//+build zalando

package backendconfig

import (
	"github.com/golang/glog"
	"github.com/zalando-techmonkeys/howler/backend"
	"github.com/zalando-techmonkeys/howler/zmon"
)

func init() {
	glog.Infof("------- REGISTERED ZALANDO\n")
	enabledBackends := []backend.Backend{zmon.Zmon{}, backend.DummyBackend{}}
	RegisteredBackends = RegisterBackends(enabledBackends)
}
