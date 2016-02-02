//+build zalando

package backendconfig

import (
	"fmt"

	"github.com/zalando-techmonkeys/howler/backend"
)

func init() {
	fmt.Printf("------- REGISTERED ZALANDO BACKEND CONFIG -------\n")
	enabledBackends := []backend.Backend{backend.Zmon{}, backend.DummyBackend{}, backend.Baboon{}, backend.Vault{}}
	RegisteredBackends = RegisterBackends(enabledBackends)
}
