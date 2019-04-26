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
    "os/exec"
    "strings"
    "path"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    editor string
)

var editCmd = &cobra.Command{
    Use:   "edit <Clip template>",
    Short: "Open Clip template in text editor",
    Long: `Open Clip template in text editor.

Clip will check the following locations for the editor to use:
  Clip config file
  Command line flag (--editor)
  $EDITOR environment variable
  Default (nano)
`,
    Args: cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        templateFilename := viper.GetString("templatedir") + "/" + os.Args[len(os.Args) - 1] + ".yml"
        err := openClipTemplateInEditor(templateFilename)
        if err != nil {
            fmt.Printf("Failed to open Clip template for editing: %v\n", err)
        }
    },
}

func init() {
    rootCmd.AddCommand(editCmd)

    // command Line flags
    editCmd.Flags().StringVarP(&editor, "editor", "e", "", "location of template directory (default is $HOME/clip)")

    // use viper to bind config to CLI flags
    viper.BindPFlag("editor", editCmd.Flags().Lookup("editor"))
}

func openClipTemplateInEditor(filename string) error {
    var editor string

    // check if clip template exists yet. if it doesn't, make it
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        err = writeTemplateFile(filename)
        if err != nil {
            return fmt.Errorf("Call to create Clip template file failed: %v\n", err)
        }
    }

    if viper.GetString("editor") != "" {
        editor = viper.GetString("editor")
        _, err := exec.LookPath(editor)
        if err != nil {
            return fmt.Errorf("Could not find an editor named '%s' in your PATH: %v\n", editor, err)
        }
    } else if os.Getenv("EDITOR") != "" {
        editor = os.Getenv("EDITOR")
        _, err := exec.LookPath(editor)
        if err != nil {
            return fmt.Errorf("Could not find an editor named '%s' in your PATH: %v\n", editor, err)
        }
    } else {
        return fmt.Errorf("No editor defined!")
    }

    // build command to run
    cmd := exec.Command(editor, filename)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("Failed to open Clip template '%s' in %s: %v\n", strings.TrimSuffix(path.Base(filename), filepath.Ext(filename)), editor, err)
    }

    return nil
}
