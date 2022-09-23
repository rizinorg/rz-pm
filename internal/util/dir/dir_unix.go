// +build darwin freebsd openbsd

package dir

import (
	"os"
)

func homePath() string {
	return os.Getenv("HOME")
}
