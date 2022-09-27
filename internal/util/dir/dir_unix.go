//go:build darwin || freebsd || openbsd || netbsd
// +build darwin freebsd openbsd netbsd

package dir

import (
	"os"
)

func homePath() string {
	return os.Getenv("HOME")
}
