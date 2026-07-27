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

// Package template reads clip's template files off disk and renders their
// text through text/template with the sprout function library loaded. Global
// template vars come in as a [Render] parameter; the package knows nothing
// about config.
package template

import (
	"bytes"
	"fmt"
	stdmaps "maps"
	"os"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/registry/checksum"
	"github.com/go-sprout/sprout/registry/conversion"
	"github.com/go-sprout/sprout/registry/crypto"
	"github.com/go-sprout/sprout/registry/encoding"
	"github.com/go-sprout/sprout/registry/env"
	"github.com/go-sprout/sprout/registry/filesystem"
	"github.com/go-sprout/sprout/registry/maps"
	"github.com/go-sprout/sprout/registry/numeric"
	"github.com/go-sprout/sprout/registry/random"
	"github.com/go-sprout/sprout/registry/reflect"
	"github.com/go-sprout/sprout/registry/regexp"
	"github.com/go-sprout/sprout/registry/semver"
	"github.com/go-sprout/sprout/registry/slices"
	"github.com/go-sprout/sprout/registry/std"
	"github.com/go-sprout/sprout/registry/strings"
	"github.com/go-sprout/sprout/registry/time"
	"github.com/go-sprout/sprout/registry/uniqueid"
	"gopkg.in/yaml.v3"
)

// File is a clip template as stored on disk: tags for filtering, the
// template's own vars, and its templated text.
type File struct {
	Tags []string `yaml:"tags"`

	Template struct {
		Vars map[string]string `yaml:"vars"`

		Text string `yaml:"text"`
	} `yaml:"template"`
}

// Load reads and unmarshals the YAML template file at filename.
func Load(filename string) (File, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return File{}, err
	}

	var tmpl File
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return File{}, err
	}

	return tmpl, nil
}

// Render executes tmpl's text with the sprout registries loaded. The template
// sees globalVars (clip's `vars` config, may be nil) with tmpl's own vars
// merged over the top, so a template var wins over a global of the same name.
func Render(tmpl File, globalVars map[string]string) (string, error) {
	vars := make(map[string]string, len(globalVars)+len(tmpl.Template.Vars))
	stdmaps.Copy(vars, globalVars)
	stdmaps.Copy(vars, tmpl.Template.Vars)

	handler := sprout.New()
	if err := handler.AddRegistries(
		checksum.NewRegistry(),
		conversion.NewRegistry(),
		crypto.NewRegistry(),
		encoding.NewRegistry(),
		env.NewRegistry(),
		filesystem.NewRegistry(),
		maps.NewRegistry(),
		numeric.NewRegistry(),
		random.NewRegistry(),
		reflect.NewRegistry(),
		regexp.NewRegistry(),
		semver.NewRegistry(),
		slices.NewRegistry(),
		std.NewRegistry(),
		strings.NewRegistry(),
		time.NewRegistry(),
		uniqueid.NewRegistry(),
	); err != nil {
		return "", fmt.Errorf("failed to add sprout registries to handler: %w", err)
	}

	t, err := template.New("clip").Funcs(handler.Build()).Parse(tmpl.Template.Text)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return out.String(), nil
}
