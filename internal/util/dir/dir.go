package dir

import (
	"os"
	"path/filepath"
)

const (
	orgSubDir     = "rizin"
	SiteDirEnvVar = "RZPM_SITEDIR"
)

func RzDir() string {
	return orgSubdDir()
}

func SiteDir() string {
	if envVar := os.Getenv(SiteDirEnvVar); envVar != "" {
		return envVar
	}

	return filepath.Join(orgSubdDir(), "rz-pm")
}

func orgSubdDir() string {
	return filepath.Join(platformPrefix(), orgSubDir)
}
