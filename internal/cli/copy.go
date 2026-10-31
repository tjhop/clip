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
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/clipboard"
	"github.com/tjhop/clip/internal/config"
	"github.com/tjhop/clip/internal/template"
)

var (
	copyCmdName  string
	copyTemplate string
)

func registerCopyCmd(app *kingpin.Application) {
	cmd := app.Command("copy", "Copy a Clip template/stdin to your clipboard.").Alias("load").Alias("in")
	cmd.HelpLong("Copy a Clip template or command output from stdin to your clipboard. With no template argument, stdin is read until EOF and copied verbatim.")
	cmd.Arg("template", "Name of the Clip template to copy.").StringVar(&copyTemplate)
	copyCmdName = cmd.FullCommand()
}

func runCopy(cfg config.Config) error {
	if copyTemplate == "" {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("could not read stdin: %w", err)
		}
		if err := clipboard.Copy(input); err != nil {
			return fmt.Errorf("could not write stdin to clipboard: %w", err)
		}
		return nil
	}

	path, err := findTemplate(cfg.TemplateDir, copyTemplate)
	if err != nil {
		return err
	}

	tmpl, err := template.Load(path)
	if err != nil {
		return fmt.Errorf("could not load Clip template '%s': %w", copyTemplate, err)
	}

	rendered, err := template.Render(tmpl, cfg.Vars)
	if err != nil {
		return fmt.Errorf("could not render Clip template '%s': %w", copyTemplate, err)
	}

	if err := clipboard.Copy([]byte(rendered)); err != nil {
		return fmt.Errorf("could not write Clip template '%s' to clipboard: %w", copyTemplate, err)
	}

	return nil
}
