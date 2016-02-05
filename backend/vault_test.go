package backend

import (
	"testing"
)

func Test_VaultAuthenticate(t *testing.T) {
	_, err := vaultAuthenticate("testURL")
	if err == nil {
		t.FailNow()
	}
}

func Test_generatePolicyTemplate(t *testing.T) {
	vb := vaultBackend{}
	policy := generatePolicyTemplate()
	if policy == "" {
		t.FailNow()
	}
}
