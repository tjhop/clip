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

package helpers

import (
	"bytes"
	"fmt"
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
	"github.com/spf13/viper"
)

func ExecuteTemplate(tmpl TemplateFile) (string, error) {
	var gotmpl bytes.Buffer
	varmap := make(map[string]string)

	// add default variables from config file to varmap
	for k, v := range viper.GetStringMapString("vars") {
		varmap[k] = v
	}
	// merge template vars to default vars from config file
	for k, v := range tmpl.Template.Vars {
		varmap[k] = v
	}

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
		return "", fmt.Errorf("Failed to add sprout registries to handler: %s\n", err.Error())
	}

	t := template.Must(template.New("Clip Template").Funcs(handler.Build()).Parse(tmpl.Template.Text))
	err := t.Execute(&gotmpl, varmap)
	if err != nil {
		return "", fmt.Errorf("Failed to execute template: %v\n", err)
	}

	return gotmpl.String(), nil
}
