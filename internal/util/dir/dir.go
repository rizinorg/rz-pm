package dir

import (
	"os"
	"path/filepath"
)

const (
	orgSubDir     = "RizinOrg"
	SiteDirEnvVar = "RZPM_SITEDIR"
)

func RzDir() string {
	return filepath.Join(orgSubdDir(), "rizin")
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
