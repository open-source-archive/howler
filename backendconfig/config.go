package backendconfig

import (
	"github.com/golang/glog"
	"github.com/zalando-techmonkeys/howler/backend"
)

var RegisteredBackends []backend.Backend

func RegisterBackends(enabledBackends []backend.Backend) []backend.Backend {
	var backends []backend.Backend
	for _, backendInstance := range enabledBackends {
		err := backendInstance.Register()
		if err != nil {
			glog.Fatalf("unable to register backend %s", backendInstance)
		}

		backends = append(backends, backendInstance)
	}
	return backends
}
