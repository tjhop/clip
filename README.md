# Clip - A Templated Clipboard Manager

## Overview
Clip is a CLI tool to let you manage templates and copy them to your clipboard so you can easily paste them into other forms or software. Clip templates can be templated using [Go templates](https://golang.org/pkg/text/template/), and they can be assigned tags to organize and filter your templates. Variables to be used in the templates can be either embedded directly into the template or put into the `clip` config file so they're available to all templates. Examples of the [config file](#Configuration) and [template files](#Templates) can be found below.

## Usage
Clip has several subcommands available, which provide the ability to create and interact with clip templates as well as the system's clipboard. Several commands also have aliases registered, so, for instance, `clip add $test` is a valid alias for `clip create $test`.

By default, if run without any subcommands or arguments, running `clip` will run the `clip list` subcommand to list available templates. If run with a single argument (ie, a clip template whose name is not registered as a subcommand), clip will run the `clip copy` command to copy the template.

You can view all available commands with `clip help`, as well as get help from each specific subcommand with `clip $command --help`:
```shell
~ $ clip help
Clip is a CLI tool to build and manage templated snippets and
interact with the systems's clipboard

Usage:
  clip [flags]
  clip [command]

Available Commands:
  copy        Copy a Clip template to your clipboard (default if just running `clip $arg`)
  create      Create a new Clip template
  edit        Open Clip template in text editor
  help        Help about any command
  list        List available Clip templates/tags (default if just running `clip`)
  paste       Print clipboard contents to stdout
  remove      Remove a Clip template
  rename      Rename a Clip template
  show        Show the raw Clip template file
  version     Print Clip build info

Flags:
      --config string        config file (default is $HOME/.clip.yml)
  -h, --help                 help for clip
  -t, --templatedir string   location of template directory (default is $HOME/clip)
  -v, --version              clip version and build info

Use "clip [command] --help" for more information about a command.
```

## Configuration
Clip uses a single configuration file (by default located at `$HOME/.clip.yml`). A basic config file will be created for you when you first run the command, but there are currently only 3 required keys:
```yml
editor: nano
templatedir: /your/home/directory/clip
vars:
  name: Clip User
```
Variables defined in the config file are global and available to all clip templates. If the variable is also defined in the template, the template version will override the global.

Currently, you'll need to edit this config file directly to change these default values.

Template configuration can be done almost entirely through the `clip` CLI and it's subcommands (create, edit, remove, rename, list, etc)

## Templates
Clip templates are YAML files with [Golang templated](https://golang.org/pkg/text/template/) text snippets and variables to use for substitutions in the template. Templates exist in a directory managed by Clip. By default, the template directory is `$HOME/clip/`, but this can be changed by editing the `templatedir` setting in the config file or passing the `--templatedir` flag at runtime.

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

For testing purposes, binaries can be built manually, as well:
    ```shell
    COMMIT=$(git rev-parse --short HEAD);
    TAG=$(git describe --always --tags --abbrev=0 | tr -d "[v\r\n]");
    go build -o clip -ldflags="-X github.com/tjhop/clip/cmd.builddate=$(date --iso-8601=seconds)
        -X github.com/tjhop/clip/cmd.version=$TAG
        -X github.com/tjhop/clip/cmd.commit=$COMMIT"
    ```

### Releases
Releases are handled by [goreleaser and github actions](https://goreleaser.com/ci/actions/).

In order to cut a release, a new semver tag must be pushed to the repo:
    Note: It's highly recommended to install [SVU](https://github.com/caarlos0/svu) to help with tag creation.

    ```shell
    git checkout master
    git fetch --tags origin master
    git tag $(svu next)
    git push --tags upstream master
    ```

## TODO
- [ ] allow editing Clip config directly through `clip` commands like template files?
- [ ] allow using different config file locations (viper has the ability to search config paths, I just couldn't think of other places I'd want the config during development)
- [X] figure out how to post release binaries on github (never done it before ¯\\\_(ツ)_/¯)
- [X] figure out how to report version/commit info that I'm bothering to embed in the build
- [X] fix reading environment var for EDITOR so it actually overrides config defaults
- [X] convert all manual filepath building to use `path.Join`
- [X] add ability copy from stdin? would allow ability to pipe output from commands to `clip copy`, which may be a nice abstraction to having to remember OS-specific commands
- [ ] add tests

## Credits/Thanks
Clip is written using the [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) libraries, with the clipboard management provided by [atotto/clipboard library](https://github.com/atotto/clipboard). They made my life a heck of a life easier, so thanks to them <3.
