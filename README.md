# rz-pm: rizin package manager

This tool aims to be a cross platform package manager for the reverse engineering
framework [Rizin](https://github.com/rizinorg/rizin).

It is still a work in progress. Any help is highly appreciating, starting from
reporting bugs and feature requests to implementing code fixes.

## How to install

Download the rz-pm binary for your system on the [latest release page](https://github.com/rizinorg/rz-pm/releases/latest). Make the file executable and you are ready to go!


| CI | Badges/URL |
|----------|---------------------------------------------------------------------|
| **GithubCI**  | [![Go](https://github.com/rizinorg/rz-pm/actions/workflows/go.yml/badge.svg)](https://github.com/rizinorg/rz-pm/actions/workflows/go.yml) |

# Available packages

The official database is available [here](https://github.com/rizinorg/rz-pm-db). You can change the repo by supplying a custom `RZPM_DB_REPO_URL` environment flag like this:

```
$ RZPM_DB_REPO_URL=https://github.com/bunnyfoofoo/my-custom-rz-pm-db rz-pm install rz-custom-plugin
```

Furthermore, to aid with debugging, you can disable auto-updating the `rz-pm-db` upon each command execution by adding `-update-db=false` flag, like this:

```
$ rz-pm -update-db=false install rz-custom-plugin
```

## Package example

```yaml
name: jsdec
version: 0.4.0
summary: Converts asm to pseudo-C code
description: Longer description of the jsdec package
source:
  url: https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz
  hash: 5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d
  directory: jsdec-0.4.0/p
  build_system: meson
  build_arguments:
    - -Djsc_folder=..
```

## Note on vendored dependencies

We use the `vendor` directory to store vendored dependencies. This allows us to just git clone the repository and have all the dependencies available.

To update the vendored dependencies, you can use the `make update-deps` command. This will first update the `go.mod` and `go.sum` files, and then copy the dependencies to the `vendor` directory.
