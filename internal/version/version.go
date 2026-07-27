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

// Package version exposes clip's build-time metadata and renders it for
// `clip --version` and the `clip version` subcommand. Modeled after
// github.com/prometheus/common/version.
package version

import (
	"cmp"
	"fmt"
	"runtime"
	"runtime/debug"
)

// Build-time metadata, injected with -ldflags by release builds. Builds
// without the injection (go install, go build) fall back to what the Go
// toolchain recorded in the binary.
var (
	Version   string
	BuildDate string
	Commit    string
)

// Print returns a human-readable, multi-line summary of the binary's build
// metadata, without a trailing newline.
func Print(programName string) string {
	v, date, commit := Version, BuildDate, Commit
	if info, ok := debug.ReadBuildInfo(); ok {
		v = cmp.Or(v, info.Main.Version)
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = cmp.Or(commit, s.Value)
			case "vcs.time":
				date = cmp.Or(date, s.Value)
			}
		}
	}

	return fmt.Sprintf("%s build info:\n\tversion: %s\n\tbuild date: %s\n\tcommit: %s\n\tgo version: %s",
		programName, v, date, commit, runtime.Version())
}
