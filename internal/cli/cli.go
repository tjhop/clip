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

// Package cli is clip's command line interface: the kingpin app, config
// resolution, and command dispatch.
package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/config"
	"github.com/tjhop/clip/internal/version"
)

const programName = "clip"

// completionFlag is the hidden flag kingpin's completion scripts call clip
// with; the partial word being completed comes along as a positional token.
const completionFlag = "--completion-bash"

// templateExts are the extensions recognized as templates. New templates get
// the first.
var templateExts = []string{".yml", ".yaml"}

var errTemplateNotFound = errors.New("clip template not found")

// App-level flag values. Empty means the flag was not given.
var (
	flagConfigFile  string
	flagTemplateDir string
)

// NewApp builds the kingpin application with all commands registered.
func NewApp() *kingpin.Application {
	// kingpin expands "@file" args into the file's contents by default;
	// template names are never files.
	kingpin.EnableFileExpansion = false

	app := kingpin.New(programName, "CLI tool to build and manage templated snippets and interact with the system's clipboard.")
	app.Version(version.Print(programName)) // VersionFlag only exists after Version()
	app.VersionFlag.Short('v')
	app.HelpFlag.Short('h')
	app.UsageWriter(os.Stdout)

	app.Flag("config", "Path to clip config file (default is $HOME/.clip.yml).").StringVar(&flagConfigFile)
	app.Flag("template-dir", "Path to the clip template directory (default is $HOME/clip).").Short('t').StringVar(&flagTemplateDir)
	app.Flag("templatedir", "Compatibility alias for --template-dir.").Hidden().StringVar(&flagTemplateDir)

	registerCopyCmd(app)
	registerCreateCmd(app)
	registerEditCmd(app)
	registerListCmd(app)
	registerPasteCmd(app)
	registerRemoveCmd(app)
	registerRenameCmd(app)
	registerShowCmd(app)
	registerVersionCmd(app)

	return app
}

// CompletionArgs drops the word under the cursor from a shell completion
// request and returns any other argument list unchanged. kingpin's completion
// scripts pass the words typed so far, partial word last, and filter whatever
// clip prints themselves. kingpin only offers command names when it reaches
// the end of the arguments without hitting a positional token (its
// default-command handling throws the alternatives away once it has tried to
// match one), so the partial word has to go before parsing.
func CompletionArgs(args []string) []string {
	if len(args) == 0 || !slices.Contains(args, completionFlag) {
		return args
	}
	if last := args[len(args)-1]; last != completionFlag && !strings.HasPrefix(last, "-") {
		return args[:len(args)-1]
	}
	return args
}

// Run resolves the config, creating the config file and template directory
// on first run, and dispatches the selected command.
func Run(cmdName string) error {
	cfg, err := resolveConfig()
	if err != nil {
		return err
	}

	switch cmdName {
	case copyCmdName:
		return runCopy(cfg)
	case createCmdName:
		return runCreate(cfg)
	case editCmdName:
		return runEdit(cfg)
	case listCmdName:
		return runList(cfg)
	case pasteCmdName:
		return runPaste()
	case removeCmdName:
		return runRemove(cfg)
	case renameCmdName:
		return runRename(cfg)
	case showCmdName:
		return runShow(cfg)
	case versionCmdName:
		return runVersion()
	default:
		return fmt.Errorf("unknown command: %s", cmdName)
	}
}

// resolveConfig loads the config file, creating it and the template directory
// on first run, and applies [config.Resolve]. First-run notices go to stderr
// so they never mix into command output.
func resolveConfig() (config.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return config.Config{}, fmt.Errorf("could not determine home directory: %w", err)
	}

	cfgPath := flagConfigFile
	if cfgPath == "" {
		cfgPath = config.DefaultPath(home)
	}

	created, err := config.EnsureConfigFile(cfgPath)
	if err != nil {
		return config.Config{}, err
	}
	if created {
		fmt.Fprintln(os.Stderr, "Clip config file not found; wrote a default one to:", cfgPath)
	}

	fileCfg, err := config.Load(cfgPath)
	if err != nil {
		return config.Config{}, err
	}

	// editEditor is the edit command's flag, but every precedence rule lives
	// here.
	cfg := config.Resolve(config.Flags{Editor: editEditor, TemplateDir: flagTemplateDir}, os.Getenv, fileCfg, home)

	// Only commands that touch the template dir need it: report the failure
	// and let those commands fail on it.
	if created, err := config.EnsureTemplateDir(cfg.TemplateDir); err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	} else if created {
		fmt.Fprintln(os.Stderr, "Template directory created at:", cfg.TemplateDir)
	}

	return cfg, nil
}

// templatePath returns where a template named name is created with the given
// extension. Names are paths relative to the template dir ("sub/foo" lives in
// a subdirectory); names that would escape it are rejected.
func templatePath(dir, name, ext string) (string, error) {
	if name == "" || !filepath.IsLocal(name) {
		return "", fmt.Errorf("invalid Clip template name '%s'", name)
	}
	return filepath.Join(dir, name+ext), nil
}

// findTemplate returns the path of the existing template named name, trying
// each extension in order. A missing template wraps errTemplateNotFound.
func findTemplate(dir, name string) (string, error) {
	for _, ext := range templateExts {
		path, err := templatePath(dir, name, ext)
		if err != nil {
			return "", err
		}

		_, err = os.Stat(path)
		if err == nil {
			return path, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("could not stat Clip template '%s': %w", name, err)
		}
	}

	return "", fmt.Errorf("%w: '%s' in %s", errTemplateNotFound, name, dir)
}
