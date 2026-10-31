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

// Package main is the clip command line tool.
package main

import (
	"os"
	"strconv"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/cli"
	"github.com/tjhop/clip/internal/clipboard"
)

func main() {
	app := cli.NewApp()

	// nilaway thinks os.Args can be nil.
	args := os.Args
	if len(args) > 0 {
		args = args[1:]
	}

	// The clipboard helper is clip re-executed by `clip copy` with the byte
	// count it is about to receive. It never goes through kingpin, so it stays
	// out of help and completion and reads no config.
	if len(args) == 2 && args[0] == clipboard.ServeCommandName {
		size, err := strconv.Atoi(args[1])
		if err != nil || size < 0 {
			app.Fatalf("%s: bad byte count %q", clipboard.ServeCommandName, args[1])
		}
		if err := clipboard.Serve(os.Stdin, size, os.NewFile(3, "ready")); err != nil {
			app.Fatalf("%v", err)
		}
		return
	}

	cmdName := kingpin.MustParse(app.Parse(cli.CompletionArgs(args)))

	if err := cli.Run(cmdName); err != nil {
		// Same "clip: error:" prefix as kingpin's own parse errors.
		app.Fatalf("%v", err)
	}
}
