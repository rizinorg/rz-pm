# rz-pm: rizin package manager

This tool is a cross platform package manager for the reverse engineering
framework [rizin](https://github.com/rizinorg/rizin).

It is a rewrite in Go of the [original Shell rz-pm script](https://github.com/rizinorg/rizin/blob/master/binrz/rz-pm/rz-pm).

This tool is still a work in progress.

| CI | Badges/URL |
|----------|---------------------------------------------------------------------|
| **GithubCI**  | [![Tests Status](https://github.com/rizinorg/rz-pm/workflows/Go/badge.svg)](https://github.com/rizinorg/rz-pm/actions?query=workflow%3AGo)|

## Package example

The official database is available [here](https://github.com/rizinorg/rz-pm-db).

```yaml
name: jsdec
type: git
repo: https://github.com/rizinorg/jsdec
desc: "[rz-native] converts asm to pseudo-C code."

install:
  - make -C p

uninstall:
  - make -C p uninstall
```
