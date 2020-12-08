package site

import "os"

// UninstallRizin removes the directory containing the rizin installation and
// all the files it contains.
func (s Site) UninstallRizin(prefix string) error {
	return os.RemoveAll(prefix)
}
