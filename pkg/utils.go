package pkg

import "strings"

func GetMajorMinorVersion(version string) string {
	splits := strings.SplitN(version, ".", 3)
	return splits[0] + "." + splits[1]
}
