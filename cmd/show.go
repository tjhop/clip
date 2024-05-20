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
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showCmd = &cobra.Command{
	Use:     "show <Clip template>",
	Aliases: []string{"cat", "dump"},
	Short:   "Show the raw Clip template file",
	Long:    `Show the output of the raw clip template file (pretty much just cat the file)`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateFilename := filepath.Join(viper.GetString("templatedir"), os.Args[len(os.Args)-1]+".yml")
		err := showClipTemplate(templateFilename)
		if err != nil {
			fmt.Printf("Call to show template failed: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func showClipTemplate(filename string) error {
	// check if template file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("Couldn't find a clip template with the name: '%s'\n", strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	} else {
		buf, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("Failed to read template file: %v\n", err)
		}

		fmt.Println(string(buf))
	}

	return nil
}
