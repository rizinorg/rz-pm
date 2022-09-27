# rz-pm: rizin package manager

This tool aims to be a cross platform package manager for the reverse engineering
framework [Rizin](https://github.com/rizinorg/rizin).

It is still a work in progress and currently it works on *NIX systems only. Any help is highly appreciating, starting from reporting bugs and feature requests to implementing code fixes.

| CI | Badges/URL |
|----------|---------------------------------------------------------------------|
| **GithubCI**  | [![Go](https://github.com/rizinorg/rz-pm/actions/workflows/go.yml/badge.svg)](https://github.com/rizinorg/rz-pm/actions/workflows/go.yml) |

# Available packages
The official database is available [here](https://github.com/rizinorg/rz-pm-db).

## Package example

```yaml
name: jsdec
version: 0.4.0
description: Converts asm to pseudo-C code
source:
  url: https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz
  hash: 5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d
  directory: jsdec-0.4.0/p
  build_system: meson
  build_arguments:
    - -Djsc_folder=..
```
