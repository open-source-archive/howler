package backend

import (
	"fmt"
	"testing"
)

func Test_isStringValid(t *testing.T) {
	bool := isStringSafe("tm")
	if !bool {
		fmt.Println("Valid string not recognized")
		t.FailNow()
	}
	bool = isStringSafe("team-techmonkeys")
	if !bool {
		fmt.Println("Valid string not recognized")
		t.FailNow()
	}
	bool = isStringSafe("}\npath \"secret/*\" {\\n\\tpolicy = \"sudo\"\\n}")
	if bool {
		fmt.Println("NOT Valid string recognized as valid")
		t.FailNow()
	}
}

func Test_buildTemplate(t *testing.T) {
	appID := "howler"
	teamID := "tm"
	tpl, err := buildTemplate(teamID, appID)
	if err != nil {
		fmt.Println("VALID parameters are failing")
		t.FailNow()
	}
	if tpl.appID != appID {
		fmt.Printf("Expected: %s, got: %s\n", appID, tpl.appID)
		t.FailNow()
	}
	if tpl.teamID != teamID {
		fmt.Printf("Expected: %s, got: %s\n", teamID, tpl.teamID)
		t.FailNow()
	}

	wrongTeamID := "}\npath \"secret/*\" {\\n\\tpolicy = \"sudo\"\\n}"
	tpl, err = buildTemplate(wrongTeamID, appID)
	if err == nil {
		fmt.Println("INVALID parameters are accepted")
		t.FailNow()
	}
}
