package dir

import (
	"os"
	"path/filepath"
)

const (
	SiteDirEnvVar = "RZPM_SITEDIR"
)

func SiteDir() string {
	if envVar := os.Getenv(SiteDirEnvVar); envVar != "" {
		return envVar
	}

	return filepath.Join(homePath(), "rz-pm")
}

func HomeDir() string {
	return homePath()
}
