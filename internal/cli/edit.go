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

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/config"
)

var (
	editCmdName  string
	editTemplate string
	editEditor   string
)

func registerEditCmd(app *kingpin.Application) {
	cmd := app.Command("edit", "Open Clip template in text editor.")
	cmd.HelpLong(`Open Clip template in text editor. The template is created first if it
doesn't exist yet.

Clip resolves the editor to use in order of precedence:
  Command line flag (--editor)
  $EDITOR environment variable
  Clip config file ("editor" key)
  Default (nano)`)
	cmd.Flag("editor", "Editor to open the Clip template with.").Short('e').StringVar(&editEditor)
	cmd.Arg("template", "Name of the Clip template to edit.").Required().StringVar(&editTemplate)
	editCmdName = cmd.FullCommand()
}

func runEdit(cfg config.Config) error {
	// The editor may carry args ("code --wait"); split it like a shell would.
	argv := strings.Fields(cfg.Editor)
	if len(argv) == 0 {
		return errors.New("no editor configured")
	}
	if _, err := exec.LookPath(argv[0]); err != nil {
		return fmt.Errorf("could not find editor '%s' in PATH: %w", argv[0], err)
	}

	path, err := findTemplate(cfg.TemplateDir, editTemplate)
	switch {
	case errors.Is(err, errTemplateNotFound):
		if path, err = templatePath(cfg.TemplateDir, editTemplate, templateExts[0]); err != nil {
			return err
		}
		if err := createTemplateFile(path); err != nil {
			return fmt.Errorf("could not create Clip template '%s': %w", editTemplate, err)
		}
		fmt.Printf("Clip template '%s' created\n", editTemplate)
	case err != nil:
		return err
	}

	cmd := exec.Command(argv[0], append(slices.Clone(argv[1:]), path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor '%s' failed on Clip template '%s': %w", cfg.Editor, editTemplate, err)
	}

	return nil
}
