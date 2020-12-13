// +build darwin freebsd

package dir

import (
	"os"
	"path/filepath"
	"testing"
)

func testSiteDir(t *testing.T) {
	if s := SiteDir(); s != filepath.Join(os.Getenv("HOME"), "bin", "prefix", "rizin", "rz-pm") {
		t.Fatal(s)
	}
}
