package pkg

type RizinPackage struct {
	Name        string
	Repo        string
	Description string `yaml:"desc"`
}

func (rp RizinPackage) Download(path string) error {
	return nil
}

func (rp RizinPackage) Install(installPath string) error {
	return nil
}
