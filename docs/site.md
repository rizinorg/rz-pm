# Site

The `rz-pm` *site* is a directory that is only managed by the software.
The user should never touch its contents manually.
The site is initialized using the `rz-pm init` command.


## Location

For each supported operating system:

- Linux:
  - `${HOME}/rz-pm`
- BSD (including macOS):
  - `${HOME}/rz-pm`
- Windows
  - `${HOMEPATH}/rz-pm`

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
