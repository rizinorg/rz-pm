package dir

import "os"

func homePath() string {
	return os.Getenv("HOMEPATH")
}
