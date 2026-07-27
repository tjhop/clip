// Copyright © 2019 TJ Hoplock <t.hoplock@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package config is clip's configuration: the YAML config file, its first-run
// creation, and the precedence rules that fold flags, environment, and file
// values into one [Config]. Precedence is hand-rolled rather than delegated to
// a config library so it stays small and testable.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultEditor   = "nano"
	configFileName  = ".clip.yml"
	templateDirName = "clip"
)

// scaffold is the config file written on first run. templatedir stays
// commented out: pinning an absolute path here would break a config file
// synced between machines.
const scaffold = `# clip configuration. Every key is optional.
#
# editor: command used by 'clip edit'. The --editor flag and $EDITOR take
#   precedence over this value. Defaults to nano.
# templatedir: directory clip reads and writes templates in. The -t /
#   --template-dir flag takes precedence. A leading ~/ expands to your home
#   directory. Defaults to $HOME/clip.
# vars: global template variables, available to every template. A template's
#   own vars override these on name collisions.

editor: nano

# templatedir: ~/clip

vars:
  name: Clip User
`

// Config is clip's configuration as it appears in the config file. The YAML
// keys are clip's on-disk compatibility surface and must not be renamed.
type Config struct {
	// Editor is the command used to edit template files.
	Editor string `yaml:"editor"`

	// TemplateDir is the directory clip reads and writes templates in.
	TemplateDir string `yaml:"templatedir"`

	// Vars are global template variables, available to every template and
	// overridable by a template's own vars.
	Vars map[string]string `yaml:"vars"`
}

// DefaultPath returns the default config file path for the given home
// directory.
func DefaultPath(home string) string {
	return filepath.Join(home, configFileName)
}

// Load reads and parses the YAML config file at path. Keys absent from the
// file are left as their zero value for [Resolve] to fill in.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("could not read config file '%s': %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("could not parse config file '%s': %w", path, err)
	}

	return cfg, nil
}

// EnsureConfigFile writes the scaffold to path unless a file is already there.
// created reports whether it wrote one, for the first-run notice.
func EnsureConfigFile(path string) (created bool, err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not create config file '%s': %w", path, err)
	}

	if _, err := f.WriteString(scaffold); err != nil {
		_ = f.Close()
		return false, fmt.Errorf("could not write config file '%s': %w", path, err)
	}
	if err := f.Close(); err != nil {
		return false, fmt.Errorf("could not write config file '%s': %w", path, err)
	}

	return true, nil
}

// EnsureTemplateDir creates the template directory at path, parents included.
// created reports whether it made one, for the first-run notice; the stat is
// needed because os.MkdirAll can't answer "did I create it?".
func EnsureTemplateDir(path string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("could not stat template directory '%s': %w", path, err)
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return false, fmt.Errorf("could not create template directory '%s': %w", path, err)
	}

	return true, nil
}

// Flags carries the config-related flag values parsed by the CLI layer. An
// empty string means the flag was not set.
type Flags struct {
	// Editor is the value of the --editor flag.
	Editor string

	// TemplateDir is the value of the --template-dir flag.
	TemplateDir string
}

// Resolve layers flags, environment, and file values into the effective
// config. getenv is injected so tests don't touch the process environment;
// production passes os.Getenv. EDITOR is the only variable read.
//
// Precedence, each level falling through when empty:
//
//	Editor:      flag > $EDITOR > file > nano
//	TemplateDir: flag > file (a leading ~/ expands to home) > <home>/clip
//	Vars:        file, if it set the key to a map; otherwise the default vars
func Resolve(flags Flags, getenv func(string) string, file Config, home string) Config {
	resolved := Config{
		Editor:      cmp.Or(flags.Editor, getenv("EDITOR"), file.Editor, defaultEditor),
		TemplateDir: cmp.Or(flags.TemplateDir, expandHome(file.TemplateDir, home), filepath.Join(home, templateDirName)),
		Vars:        file.Vars,
	}

	// nil (key omitted or null) means the defaults apply; an empty map means
	// the user cleared the global vars.
	if resolved.Vars == nil {
		resolved.Vars = map[string]string{"name": "Clip User"}
	}

	return resolved
}

// expandHome replaces a leading "~/" (or a bare "~") in path with home. Only
// file values need it; the shell already expanded ~ in flag values.
func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, rest)
	}
	return path
}
