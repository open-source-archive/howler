package backendconfig

import (
	"github.com/golang/glog"
	"github.com/zalando-techmonkeys/howler/backend"
)

var RegisteredBackends []backend.Backend

func RegisterBackends(enabledBackends []backend.Backend) []backend.Backend {
	var backends []backend.Backend
	for _, backendImplementation := range enabledBackends {
		err, backendInstance := backendImplementation.Register()
		if err != nil {
			glog.Fatalf("unable to register backend %s", backendImplementation)
		}
		backends = append(backends, backendInstance)
	}
	return backends
}
