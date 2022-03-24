// +build darwin freebsd openbsd

package dir

import (
	"os"
	"path/filepath"
)

func homePath() string {
	return os.Getenv("HOME")
}
