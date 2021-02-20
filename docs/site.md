# Site

The `rz-pm` *site* is a directory that is only managed by the software.
The user should never touch its contents manually.
The site is initialized using the `rz-pm init` command.


## Location

For each supported operating system, by order of preference:

- Linux:
  - `${XDG_DATA_HOME}/rizin/rz-pm` if `$XDG_DATA_HOME` is defined;
  - `${HOME}/.local/share/rizin/rz-pm` otherwise
- BSD (including macOS): `${HOME}/Library/rizin/rz-pm`
- Windows
  - `${APPDATA}/rizin/rz-pm` if `$APPDATA` is defined
  - `${HOMEPATH}/rizin/rz-pm` otherwise

## Contents

```
$RZPM_SITE
├── installed/
│   └── pkg1.yaml
│   └── pkg2.yaml
│   └── pkg-from-cli.yaml
├── rz-pm-db/
    └── db/
        └── pkg1.yaml
        └── pkg2.yaml
```
