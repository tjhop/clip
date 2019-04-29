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
    "os"
    "strings"
    "path/filepath"
    "io/ioutil"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

const baseTemplateFileString string =
`# See README.md for detailed information
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

// createCmd represents the create command
var createCmd = &cobra.Command{
    Use:   "create <Clip template>",
    Aliases: []string{"add", "make"},
    Short: "Create a new Clip template",
    Long: `Create a Clip template. Clip templates are YAML files with embedded Go templates and variables.
`,
    Args: cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        templateFilename := filepath.Join(viper.GetString("templatedir"), os.Args[len(os.Args) - 1] + ".yml")
        err := writeTemplateFile(templateFilename)
        if err != nil {
            fmt.Printf("Call to create template failed: %v\n", err)
        }
    },
}

func init() {
    rootCmd.AddCommand(createCmd)
}

func writeTemplateFile(filename string) error {
    // create template file if it doesn't exist
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        err := ioutil.WriteFile(filename, []byte(baseTemplateFileString), 0644)
        if err != nil {
            return fmt.Errorf("Failed to create template file: %v\n", err)
        }

        fmt.Printf("Clip template '%s' created\n", strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
    } else {
        fmt.Printf("A Clip template with the name '%s' already exists\n", strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
    }

    return nil
}
