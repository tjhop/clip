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
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/config"
)

var (
	renameCmdName string
	renameOld     string
	renameNew     string
)

func registerRenameCmd(app *kingpin.Application) {
	cmd := app.Command("rename", "Rename a Clip template.").Alias("move")
	cmd.HelpLong(`Rename an existing Clip template. The Clip template must already exist,
and the new name must be available.`)
	cmd.Arg("old", "Current name of the Clip template.").Required().StringVar(&renameOld)
	cmd.Arg("new", "New name for the Clip template.").Required().StringVar(&renameNew)
	renameCmdName = cmd.FullCommand()
}

func runRename(cfg config.Config) error {
	src, err := findTemplate(cfg.TemplateDir, renameOld)
	if err != nil {
		return err
	}

	// os.Rename replaces an existing destination, so refuse up front.
	if _, err := findTemplate(cfg.TemplateDir, renameNew); err == nil {
		return fmt.Errorf("clip template '%s' already exists", renameNew)
	} else if !errors.Is(err, errTemplateNotFound) {
		return err
	}

	dst, err := templatePath(cfg.TemplateDir, renameNew, filepath.Ext(src))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("could not rename Clip template '%s' to '%s': %w", renameOld, renameNew, err)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("could not rename Clip template '%s' to '%s': %w", renameOld, renameNew, err)
	}

	fmt.Printf("Clip template '%s' renamed to '%s'\n", renameOld, renameNew)
	return nil
}
