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

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tjhop/clip/helpers"
)

// let's declare some vars
var (
	cfgFile     string // location of config file
	templateDir string // location of template directory
	showBuild   bool   // whether or not to print version info
)

// rootCmd is the bare `clip` command that cobra executes
var rootCmd = &cobra.Command{
	Use:   "clip",
	Short: "Golang template and clipboard editor",
	Long: `Clip is a CLI tool to build and manage templated snippets and
interact with the systems's clipboard`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if showBuild {
			versionCmd.Run(cmd, args)
		} else {
			if len(args) == 0 {
				// If no subcommand is provided, run `clip list` by default
				listCmd.Run(cmd, args)
			} else if len(args) == 1 {
				// If no subcommand is provided but a command line arg is
				// provided, then run `clip copy $arg` by default
				copyCmd.Run(cmd, args)
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// functions that will always be run when this app is called
	// https://godoc.org/github.com/spf13/cobra#OnInitialize
	cobra.OnInitialize(initClip)

	// config defaults
	viper.SetDefault("editor", "nano")
	viper.SetDefault("vars", map[string]interface{}{"name": "Clip User"})

	// command Line flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.clip.yml)")
	rootCmd.PersistentFlags().StringVarP(&templateDir, "templatedir", "t", "", "location of template directory (default is $HOME/clip)")
	rootCmd.Flags().BoolVarP(&showBuild, "version", "v", false, "clip version and build info")

	// use viper to bind config to CLI flags
	viper.BindPFlag("templatedir", rootCmd.PersistentFlags().Lookup("templatedir"))
}

// initClip will set config defaults, read in config file, and initialize clip template directory if it doesn't exist yet
func initClip() {
	cfgName := "clip"
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Set config file to home directory + name ".clip.yml"
		cfgFile = filepath.Join(home, "."+cfgName+".yml")
		viper.SetConfigType("yaml")
		viper.SetConfigFile(cfgFile)
	}

	if viper.GetString("templatedir") != "" {
		viper.Set("templatedir", templateDir)
	} else {
		viper.Set("templatedir", filepath.Join(home, cfgName))
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		// I should be able to use Viper to create+write the config for me if it
		// doesn't exist, but the `SafeWriteConfig` function is still broken upstream:
		// https://github.com/spf13/viper/pull/450/files

		fmt.Println("Clip config file not found; writing config file to: ", viper.ConfigFileUsed())
		// check if config file exists
		if _, err := os.Stat(viper.ConfigFileUsed()); os.IsNotExist(err) {
			err := helpers.WriteConfigFile(viper.ConfigFileUsed(), viper.AllSettings())
			if err != nil {
				fmt.Println("Call to write Clip configuration file failed: ", err)
				os.Exit(1)
			}
		}
	}

	// if template directory doesn't exist, create it
	if _, err := os.Stat(viper.GetString("templatedir")); os.IsNotExist(err) {
		err = os.MkdirAll(viper.GetString("templatedir"), 0755)
		if err != nil {
			fmt.Printf("Could not initialize template directory: %v\n", err)
		} else {
			fmt.Println("Template directory created at: " + viper.GetString("templatedir"))
		}
	}
}
