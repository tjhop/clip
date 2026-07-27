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
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/config"
)

// templateScaffold is the content of a newly created template file.
const templateScaffold = `# See README.md for detailed information
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
`

var (
	createCmdName  string
	createTemplate string
)

func registerCreateCmd(app *kingpin.Application) {
	cmd := app.Command("create", "Create a new Clip template.").Alias("add").Alias("make")
	cmd.HelpLong("Create a Clip template. Clip templates are YAML files with embedded Go templates and variables.")
	cmd.Arg("template", "Name of the Clip template to create.").Required().StringVar(&createTemplate)
	createCmdName = cmd.FullCommand()
}

func runCreate(cfg config.Config) error {
	path, err := templatePath(cfg.TemplateDir, createTemplate, templateExts[0])
	if err != nil {
		return err
	}

	if err := createTemplateFile(path); err != nil {
		return fmt.Errorf("could not create Clip template '%s': %w", createTemplate, err)
	}

	fmt.Printf("Clip template '%s' created\n", createTemplate)
	return nil
}

// createTemplateFile writes the template scaffold to path, parents included.
// A file already there is an error wrapping fs.ErrExist.
func createTemplateFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}

	if _, err := f.WriteString(templateScaffold); err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}
