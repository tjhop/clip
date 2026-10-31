# Clip - A Templated Clipboard Manager

[![license](https://img.shields.io/github/license/tjhop/clip)](https://github.com/tjhop/clip/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/tjhop/clip)](https://goreportcard.com/report/github.com/tjhop/clip)
[![Latest Release](https://img.shields.io/github/v/release/tjhop/clip)](https://github.com/tjhop/clip/releases/latest)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/tjhop/clip/total)

## Overview
Clip is a CLI tool to let you manage templates and copy them to your clipboard so you can easily paste them into other forms or software. Clip templates can be templated using [Go templates](https://golang.org/pkg/text/template/), and they can be assigned tags to organize and filter your templates. Variables to be used in the templates can be either embedded directly into the template or put into the `clip` config file so they're available to all templates. Examples of the [config file](#Configuration) and [template files](#Templates) can be found below.

## Usage
Clip has several subcommands available, which provide the ability to create and interact with clip templates as well as the system's clipboard. Several commands also have aliases registered, so, for instance, `clip add $test` is a valid alias for `clip create $test`:

| Command | Aliases |
| ------- | ------- |
| `copy` | `load`, `in` |
| `create` | `add`, `make` |
| `paste` | `out`, `print` |
| `remove` | `delete` |
| `rename` | `move` |
| `show` | `cat`, `dump` |

Running `clip` with no subcommand lists your templates (`clip list`). Copying a template is `clip copy <template>`.

With no template argument, `clip copy` reads stdin until EOF and copies it verbatim, so `some-command | clip copy` works as an OS-agnostic clipboard pipe.

On Linux and the BSDs, `clip copy` leaves a small background process (`clip serve-clipboard`) serving the text, same as `wl-copy` does: on X11 and Wayland the program that copied is the one that answers pastes, so the text would vanish the moment `clip` exited. It exits on its own once something else takes the clipboard. Compositors with the data-control protocol are used directly; the rest (GNOME included) go through XWayland, so `DISPLAY` must be set (GNOME sets it). macOS and Windows hold the data themselves, so there's no background process. The helper reads the copy back before `clip copy` returns, so a copy that can't be pasted fails with exit 1 instead of leaving a broken clipboard; on X11 (GNOME's XWayland included) that is anything over 262116 bytes, a limit of the library's X11 client.

You can view all available commands with `clip --help`, as well as get help from each specific subcommand with `clip $command --help`:
```shell
~ $ clip --help
usage: clip [<flags>] <command> [<args> ...]

CLI tool to build and manage templated snippets and interact with the system's
clipboard.


Flags:
  -h, --[no-]help      Show context-sensitive help (also try --help-long and
                       --help-man).
  -v, --[no-]version   Show application version.
      --config=CONFIG  Path to clip config file (default is $HOME/.clip.yml).
  -t, --template-dir=TEMPLATE-DIR
                       Path to the clip template directory (default is
                       $HOME/clip).

Commands:
help [<command>...]
    Show help.

copy [<template>]
    Copy a Clip template/stdin to your clipboard.

create <template>
    Create a new Clip template.

edit [<flags>] <template>
    Open Clip template in text editor.

list* [<flags>]
    List available Clip templates/tags (default if just running `clip`).

paste
    Print clipboard contents to stdout.

remove <template>
    Remove a Clip template.

rename <old> <new>
    Rename a Clip template.

show <template>
    Show the raw Clip template file.

version
    Print Clip build info.
```

The `--templatedir` spelling of the `--template-dir` flag is still accepted for backwards compatibility.

### Shell Completion
Clip can generate its own shell completion scripts:
```shell
# bash
source <(clip --completion-script-bash)

# zsh
source <(clip --completion-script-zsh)
```

### Errors and exit status
Every failure, including a missing template, is reported on stderr with exit status 1. First-run notices (config file and template directory creation) go to stderr as well, so whatever a command prints on stdout is safe to pipe.

## Configuration
Clip uses a single configuration file, `$HOME/.clip.yml` by default (override it with `--config`). A commented scaffold is written there on first run if the file is missing. Every key is optional:
```yml
editor: nano

# templatedir: ~/clip

vars:
  name: Clip User
```

| Key | Description |
| --- | ----------- |
| `editor` | Command used by `clip edit`. It may carry arguments, e.g. `code --wait`. |
| `templatedir` | Directory your templates live in. A leading `~/` expands to your home directory. Left unset, it defaults to `$HOME/clip` on whichever machine runs clip, which keeps a config file synced between machines portable. |
| `vars` | Global template variables, available to every template. A template's own `vars` override globals with the same name. |

Settings are resolved with the following precedence (highest first):

| Setting | Precedence |
| ------- | ---------- |
| config file path | `--config` flag > `$HOME/.clip.yml` |
| `editor` | `--editor` flag > `$EDITOR` environment variable > config file > default (`nano`) |
| `templatedir` | `-t`/`--template-dir` flag > config file > default (`$HOME/clip`) |
| `vars` | config file > default (`name: Clip User`) |

Template configuration can be done almost entirely through the `clip` CLI and it's subcommands (create, edit, remove, rename, list, etc)

## Templates
Clip templates are YAML files with [Golang templated](https://golang.org/pkg/text/template/) text snippets and variables to use for substitutions in the template. Templates exist in a directory managed by Clip. By default, the template directory is `$HOME/clip/`, but this can be changed by editing the `templatedir` setting in the config file or passing the `-t`/`--template-dir` flag at runtime.

A template's name is its path inside the template directory without the `.yml` (or `.yaml`) extension, so `$HOME/clip/work/standup.yml` is the template `work/standup`. `clip list` prints exactly these names, and every other command accepts them; `clip create work/standup` creates the subdirectory as needed.

Clip also imports the [sprout template function library](https://docs.atom.codes/sprout) and loads functions from all registries _except the `backward` registry which contains deprecated functions_.

The base template that gets created is pretty simple:
```yml
# See README.md for detailed information
#
# Example template:
#
# tags:
#   - personal
#
# template:
#   vars:
#     value: Hello, world!
#   text: |
#     The value of the variable is: "{{ .value }}"

tags: []

template:
  vars: {}
  text: |
```
What the keys do:

| Key | Description | Configuration |
| --- | ----------- | ------------- |
| `tags` | Metadata tags that you'd like to apply to this template (purely for your organizational needs) | List |
| `template:vars` | Variables that will be available to the Go templating system when it renders your clip template. These variables override any global variables with the same name you define in the Clip config file. | Accepts an arbitrary number of `key: value` pairs to define variables and their values |
| `template:text` | The text to be rendered through Go's template system and loaded onto your clipboard | Accepts a YAML multi-line string (be careful with indentation!) |

Example template:
```shell
~ $ clip show readme-template-example
tags:
  - personal
  - side-project
  - golang

template:
  vars:
    name: "tjhop"
    greeting: "Hello, Clip User!"
    signature: "Have fun!"
  text: |
    {{ .greeting }}

    This is the general format of a Clip template file. The tags assigned to this template can be used to filter output when using the `list` command, like so:
    `clip list --tags-only`
    `clip list --tags personal,golang`

    More info can be found using the `--help` flag on any of the subcommands, too.

    {{ .signature }}
    {{ .name }}
```

After copying the template, we'll end up with the following content on our clipboard:
```shell
~ $ clip copy readme-template-example
~ $ clip paste
Hello, Clip User!

This is the general format of a Clip template file. The tags assigned to this template can be used to filter output when using the `list` command, like so:
`clip list --tags-only`
`clip list --tags personal,golang`

More info can be found using the `--help` flag on any of the subcommands, too.

Have fun!
tjhop
```

Example template interaction using functions from sprout:
```shell
# 'env' and 'default' are functions from sprout
~ -> clip show demo
template:
  text: |
    Hello, {{ env "NAME" | default "Name unset" }}!

~ -> clip copy demo
~ -> clip paste
Hello, Name unset!

~ -> NAME="McLovin" clip copy demo
~ -> clip paste
Hello, McLovin!
```

## Building
### Binaries
Binaries are built by [goreleaser and github actions](https://goreleaser.com/ci/actions/). Please see [Releases](#Releases) for more information on building binaries for release.

For local development, `devbox.json` provides the toolchain the Makefile expects (Go, golangci-lint, goreleaser); `devbox shell` drops you into it. nilaway is a `go.mod` tool dependency, so `make lint` builds it with the repo's Go toolchain. Binaries are built with the Makefile:

```bash
~ $ make

Usage:
  make <target>

Targets:
  help                           print this help message
  tidy                           tidy modules
  fmt                            apply go code style formatter
  lint                           run linters
  test                           run unit tests
  binary                         build a binary via goreleaser
  build                          alias for `binary`
```

### Releases
Releases are handled by [goreleaser and github actions](https://goreleaser.com/ci/actions/).

In order to cut a release, a new semver tag must be pushed to the repo:
    Note: It's highly recommended to install [SVU](https://github.com/caarlos0/svu) to help with tag creation.

    ```shell
    git checkout main
    git fetch --tags origin main
    git tag $(svu next)
    git push --tags upstream main
    ```

## TODO
- [ ] allow editing Clip config directly through `clip` commands like template files?
- [ ] allow searching multiple config file locations (beyond the explicit `--config` flag)
- [X] figure out how to post release binaries on github (never done it before ¯\\\_(ツ)_/¯)
- [X] figure out how to report version/commit info that I'm bothering to embed in the build
- [X] fix reading environment var for EDITOR so it actually overrides config defaults
- [X] convert all manual filepath building to use `path.Join`
- [X] add ability copy from stdin? would allow ability to pipe output from commands to `clip copy`, which may be a nice abstraction to having to remember OS-specific commands
- [X] add tests

## Credits/Thanks
Clip is written using the [kingpin](https://github.com/alecthomas/kingpin) library, with the clipboard management provided by the [golang.design/x/clipboard library](https://github.com/golang-design/clipboard). They made my life a heck of a lot easier, so thanks to them <3. Thanks also to [Cobra](https://github.com/spf13/cobra), [Viper](https://github.com/spf13/viper), and [atotto/clipboard](https://github.com/atotto/clipboard) for powering clip's first several years.
