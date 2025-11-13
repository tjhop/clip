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

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tjhop/clip/helpers"
)

var copyCmd = &cobra.Command{
	Use:     "copy <Clip template>",
	Aliases: []string{"load", "in"},
	Short:   "Copy a Clip template/Stdin to your clipboard (default if just running `clip $arg`)",
	Long:    `Copy a Clip template or command output from Stdin to your clipboard`,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := writeStdinToClipboard()
			if err != nil {
				fmt.Printf("Failed to copy from stdin: %v\n", err)
			}
		} else if len(args) == 1 {
			templateFilename := filepath.Join(viper.GetString("templatedir"), os.Args[len(os.Args)-1]+".yml")
			err := writeClipTemplateToClipboard(templateFilename)
			if err != nil {
				fmt.Printf("Failed to copy Clip template '%s' to clipboard: %v\n", strings.TrimSuffix(filepath.Base(templateFilename), filepath.Ext(templateFilename)), err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}

func writeClipTemplateToClipboard(filename string) error {
	tmpl, err := helpers.LoadTemplateFile(filename)
	if err != nil {
		return fmt.Errorf("couldn't load Clip template file '%s': %w", strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), err)
	}

	renderedTemplateString, err := helpers.ExecuteTemplate(tmpl)
	if err != nil {
		return fmt.Errorf("failed to render Go Template: %w", err)
	}

	err = clipboard.WriteAll(renderedTemplateString)
	if err != nil {
		return fmt.Errorf("failed to write Clip template to clipboard: %w", err)
	}

	return nil
}

func writeStdinToClipboard() error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	err = clipboard.WriteAll(string(input))
	if err != nil {
		return fmt.Errorf("failed to write data from stdin to clipboard: %w", err)
	}

	return nil
}
