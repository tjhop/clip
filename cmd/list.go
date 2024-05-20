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
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tjhop/clip/helpers"
)

var (
	tags     []string
	tagsOnly bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use: "list",
	Long: `List available Clip templates (can be filtered by tag) or list the available tags.

Example:
  clip list
  clip list --tags-only
  clip list --tags personal,work`,
	Short: "List available Clip templates/tags (default if just running `clip`)",
	Run: func(cmd *cobra.Command, args []string) {
		list()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// command Line flags
	listCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "comma separated list of tags to filter templates")
	listCmd.Flags().BoolVar(&tagsOnly, "tags-only", false, "list all tags used in the templates")
	listCmd.Flags().BoolVar(&tagsOnly, "list-tags", false, "alias for '--tags-only' flag")
	listCmd.Flags().BoolVar(&tagsOnly, "show-tags", false, "alias for '--tags-only' flag")
}

func list() {
	if tagsOnly {
		err := listTemplateTags(viper.GetString("templatedir"))
		if err != nil {
			fmt.Printf("Call to list Clip template tags failed: %v\n", err)
		}
	} else {
		err := listTemplates(viper.GetString("templatedir"))
		if err != nil {
			fmt.Printf("Call to list Clip templates failed: %v\n", err)
		}
	}
}

func listTemplates(dir string) error {
	var files []string

	// walk the template directory and get the files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			filenameWithoutExtension := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

			// if `--tags` filter was provided, check if file contains one of the provided tags
			if len(tags) > 0 {
				for _, tag := range tags {
					tmpl, err := helpers.LoadTemplateFile(filepath.Join(dir, info.Name()))
					if err != nil {
						return fmt.Errorf("Couldn't load Clip template '%s' to check for tags: %v\n", filenameWithoutExtension, err)
					}

					if helpers.Contains(tmpl.Tags, tag) && !helpers.Contains(files, filenameWithoutExtension) {
						files = append(files, filenameWithoutExtension)
					}
				}
			} else {
				files = append(files, filenameWithoutExtension)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Failed to walk template directory: %v\n", err)
	}

	for _, file := range files {
		fmt.Println(file)
	}

	return nil
}

func listTemplateTags(dir string) error {
	var tags []string

	// walk the template directory and get the files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			tmpl, err := helpers.LoadTemplateFile(filepath.Join(dir, info.Name()))
			if err != nil {
				return fmt.Errorf("Couldn't load Clip template to check for tags: %v\n", err)
			}

			for _, tag := range tmpl.Tags {
				if !helpers.Contains(tags, tag) {
					tags = append(tags, tag)
				}
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Failed to walk template directory: %v\n", err)
	}

	sort.Strings(tags)
	for _, tag := range tags {
		fmt.Println(tag)
	}

	return nil
}
