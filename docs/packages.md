# Packages

Packages are additions to the `rizin` software.
A package can consist in a set of source files that have to be built against the current `rizin` installation, or in pre-built binaries.

## Plugin file

A plugin is described by a plugin file.
A plugin file is a text file in the YAML format that contains various metadata about the plugin as well as instructions on how to install the package for each platform.

### Schema

```yaml
---
name: my-package  # must be unique and equals to the file name
version: 1.2.3
description: Some description
source:
  url: http://a-random.url/zip-archive.zip
  hash: sha256hash
  build_system: meson
  build_arguments:
    - -Darg1=val1
    - -Darg2=val2
artifacts:
  - os: linux
    url: http://a-random.url/zip-archive.zip
    hash: sha256hash
    bins:
      - path/bin1
      - path/bin2
    libs:
      - path/lib1
      - path/lib2
    plugins:
      - path/plugin1
      - path/plugin2
```
