//+build dummy !zalando

package backendconfig

import (
	"fmt"

	"github.com/zalando-techmonkeys/howler/backend"
)

func init() {
	fmt.Printf("------- REGISTERED DUMMY BACKEND CONFIG -------\n")
	enabledBackends := []backend.Backend{backend.DummyBackend{}}
	RegisteredBackends = RegisterBackends(enabledBackends)
}
