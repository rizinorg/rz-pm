package rzpackage

import "github.com/rizinorg/rzpm/pkg"

type fileMgr struct{}

func (fm *fileMgr) CopyFile(src, dst string) error {
	return pkg.CopyFile(src, dst)
}
