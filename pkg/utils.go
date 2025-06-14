package pkg

import "strings"

func GetMajorMinorVersion(version string) string {
	splits := strings.SplitN(version, ".", 3)
	var major, minor string

	major = splits[0]
	if len(splits) < 2 {
		// If there is no minor version, return the major version with a ".0"
		minor = "0"
	} else {
		minor = splits[1]
	}

	return major + "." + minor
}
