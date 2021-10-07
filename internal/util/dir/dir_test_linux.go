package dir

import (
	"os"
	"testing"

	"github.com/rizinorg/rz-pm/testdata"
)

func testSiteDir(t *testing.T) {
	const xdgDataHomeVar = "XDG_DATA_HOME"

	t.Run(xdgDataHomeVar+"=/tmp/test", func(t *testing.T) {
		testdata.SetEnvVar(t, xdgDataHomeVar, "/tmp/test")

		if s := SiteDir(); s != "/tmp/test/share/rz-pm" {
			t.Fatal(s)
		}
	})

	t.Run(xdgDataHomeVar+" unset", func(t *testing.T) {
		if err := os.Unsetenv(xdgDataHomeVar); err != nil {
			t.Fatalf("could not unset %s: %v", xdgDataHomeVar, err)
		}

		if s := SiteDir(); s != os.Getenv("HOME")+"/.local/share/rz-pm" {
			t.Fatal(s)
		}
	})
}
