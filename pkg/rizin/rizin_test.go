package rizin

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeRizinScript returns the contents of a bash script that mimics "rizin -H" output.
func fakeRizinScript() string {
	return `#!/bin/bash
if [[ "$1" == "-H" ]]; then
cat <<EOF
RZ_VERSION=9.9.9-fake
RZ_PREFIX=/fake/prefix
RZ_EXTRA_PREFIX=/fake/extra
RZ_MAGICPATH=/fake/magic
RZ_INCDIR=/fake/include
RZ_LIBDIR=/fake/lib
RZ_SIGDB=/fake/sigdb
RZ_EXTRA_SIGDB=/fake/extra_sigdb
RZ_LIBEXT=.so
RZ_CONFIGHOME=/fake/config
RZ_DATAHOME=/fake/data
RZ_CACHEHOME=/fake/cache
RZ_LIB_PLUGINS=/fake/plugins
RZ_EXTRA_PLUGINS=/fake/extra_plugins
RZ_USER_PLUGINS=/fake/user_plugins
RZ_IS_PORTABLE=0
EOF
else
	echo "Unknown option" >&2
	exit 1
fi
`
}

func TestGetRizinInfo_FakeExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	fakeRizinPath := filepath.Join(tmpDir, "rizin")

	// Write the fake rizin script
	if err := os.WriteFile(fakeRizinPath, []byte(fakeRizinScript()), 0755); err != nil {
		t.Fatalf("failed to write fake rizin: %v", err)
	}

	// Prepend tmpDir to PATH
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

	info, err := GetRizinInfo()
	if err != nil {
		t.Fatalf("GetRizinInfo failed: %v", err)
	}

	// Check a few fields for correctness
	if info.Version != "9.9.9-fake" {
		t.Errorf("expected Version=9.9.9-fake, got %q", info.Version)
	}
	if info.Prefix != "/fake/prefix" {
		t.Errorf("expected Prefix=/fake/prefix, got %q", info.Prefix)
	}
	if info.LibExt != ".so" {
		t.Errorf("expected LibExt=.so, got %q", info.LibExt)
	}
	if info.IsPortable != "0" {
		t.Errorf("expected IsPortable=0, got %q", info.IsPortable)
	}
}

func TestGetRizinInfo_NotInPath(t *testing.T) {
	// Remove rizin from PATH
	t.Setenv("PATH", "")

	_, err := GetRizinInfo()
	if err == nil {
		t.Fatal("expected error when rizin is not in PATH, got nil")
	}
	if want := "rizin does not seem to be installed"; err == nil || err.Error()[:len(want)] != want {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetRizinInfo_FailsToRun(t *testing.T) {
	tmpDir := t.TempDir()
	fakeRizinPath := filepath.Join(tmpDir, "rizin")

	// Write a script that always fails
	script := "#!/bin/sh\nexit 42\n"
	if err := os.WriteFile(fakeRizinPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake rizin: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

	_, err := GetRizinInfo()
	if err == nil {
		t.Fatal("expected error when rizin fails to run, got nil")
	}
}