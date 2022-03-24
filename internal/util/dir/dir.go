package dir

import (
	"os"
	"path/filepath"
)

const (
	orgSubDir     = "rizin"
	SiteDirEnvVar = "RZPM_SITEDIR"
)

func SiteDir() string {
	if envVar := os.Getenv(SiteDirEnvVar); envVar != "" {
		return envVar
	}

	return filepath.Join(platformPrefix(), "share", "rz-pm")
}
