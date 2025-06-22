package rizin

import (
	"fmt"
	"os/exec"

	"github.com/rizinorg/rz-pm/pkg/envparse"
)

type RizinInfo struct {
	Version      string `env:"RZ_VERSION"`
	Prefix       string `env:"RZ_PREFIX"`
	ExtraPrefix  string `env:"RZ_EXTRA_PREFIX"`
	MagicPath    string `env:"RZ_MAGICPATH"`
	IncDir       string `env:"RZ_INCDIR"`
	LibDir       string `env:"RZ_LIBDIR"`
	SigDB        string `env:"RZ_SIGDB"`
	ExtraSigDB   string `env:"RZ_EXTRA_SIGDB"`
	LibExt       string `env:"RZ_LIBEXT"`
	ConfigHome   string `env:"RZ_CONFIGHOME"`
	DataHome     string `env:"RZ_DATAHOME"`
	CacheHome    string `env:"RZ_CACHEHOME"`
	LibPlugins   string `env:"RZ_LIB_PLUGINS"`
	ExtraPlugins string `env:"RZ_EXTRA_PLUGINS"`
	UserPlugins  string `env:"RZ_USER_PLUGINS"`
	IsPortable   string `env:"RZ_IS_PORTABLE"`
}

func GetRizinInfo() (RizinInfo, error) {
	if _, err := exec.LookPath("rizin"); err != nil {
		return RizinInfo{}, fmt.Errorf("rizin does not seem to be installed on your system. Make sure it is installed and in PATH")
	}

	cmd := exec.Command("rizin", "-H")
	out, err := cmd.Output()
	if err != nil {
		return RizinInfo{}, fmt.Errorf("failed to run rizin: %w", err)
	}

	info := RizinInfo{}
	err = envparse.Unmarshal(string(out), &info)
	if err != nil {
		return RizinInfo{}, fmt.Errorf("failed to parse rizin info: %w", err)
	}

	return info, nil
}
