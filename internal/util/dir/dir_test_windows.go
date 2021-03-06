package dir

import (
	"testing"

	"github.com/rizinorg/rzpm/testdata"
)

func testSiteDir(t *testing.T) {
	t.Run(`APPDATA=C:\temp`, func(t *testing.T) {
		testdata.SetEnvVar(t, "APPDATA", `C:\temp`)

		if s := SiteDir(); s != `C:\temp\RizinOrg\rz-pm` {
			t.Fatal(s)
		}
	})
}
