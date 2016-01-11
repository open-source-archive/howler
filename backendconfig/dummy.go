//+build dummy !zalando

package backendconfig

import (
	"github.com/golang/glog"
	"github.com/zalando-techmonkeys/howler/backend"
)

func init() {
	glog.Infof("------- REGISTERED DUMMY\n")
	enabledBackends := []backend.Backend{backend.DummyBackend{}}
	RegisteredBackends = RegisterBackends(enabledBackends)
}
