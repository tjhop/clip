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

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/config"
)

var (
	showCmdName  string
	showTemplate string
)

func registerShowCmd(app *kingpin.Application) {
	cmd := app.Command("show", "Show the raw Clip template file.").Alias("cat").Alias("dump")
	cmd.HelpLong("Show the output of the raw clip template file (pretty much just cat the file).")
	cmd.Arg("template", "Name of the Clip template to show.").Required().StringVar(&showTemplate)
	showCmdName = cmd.FullCommand()
}

func runShow(cfg config.Config) error {
	path, err := findTemplate(cfg.TemplateDir, showTemplate)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read Clip template '%s': %w", showTemplate, err)
	}

	_, err = os.Stdout.Write(data)
	return err
}
