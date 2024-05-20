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

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:     "rename <old> <new>",
	Aliases: []string{"move"},
	Short:   "Rename a Clip template",
	Long: `Rename an existing Clip template. The Clip template must already exist,
and the new name must be available.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sourceTemplateFilename := filepath.Join(viper.GetString("templatedir"), os.Args[len(os.Args)-2]+".yml")
		destinationTemplateFilename := filepath.Join(viper.GetString("templatedir"), os.Args[len(os.Args)-1]+".yml")
		err := renameTemplateFile(sourceTemplateFilename, destinationTemplateFilename)
		if err != nil {
			fmt.Printf("Call to rename template failed: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}

func renameTemplateFile(sourceFilename, destinationFilename string) error {
	// check to ensure source template exists
	if _, err := os.Stat(sourceFilename); os.IsNotExist(err) {
		fmt.Printf("Couldn't find a Clip template with the name: '%s'\n", strings.TrimSuffix(filepath.Base(sourceFilename), filepath.Ext(sourceFilename)))
	}

	// check to ensure destination template does not exist
	if _, err := os.Stat(destinationFilename); err == nil {
		fmt.Printf("No action taken: Found an existing Clip template with the name: '%s'\n", strings.TrimSuffix(filepath.Base(destinationFilename), filepath.Ext(destinationFilename)))
	}

	err := os.Rename(sourceFilename, destinationFilename)
	if err != nil {
		return fmt.Errorf("Failed to rename clip template file: %v\n", err)
	}

	return nil
}
