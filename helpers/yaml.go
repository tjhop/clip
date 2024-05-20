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
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type TemplateFile struct {
	Tags []string `yaml:"tags"`

	Template struct {
		Vars map[string]string `yaml:"vars"`

		Text string `yaml:"text"`
	} `yaml:"template"`
}

func LoadTemplateFile(filename string) (TemplateFile, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return TemplateFile{}, err
	}

	var tmpl TemplateFile
	err = yaml.Unmarshal(bytes, &tmpl)
	if err != nil {
		return TemplateFile{}, err
	}

	return tmpl, nil
}

func WriteConfigFile(filename string, data interface{}) error {
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("Could not marshal data:\n%#v\n...to yaml: %v", data, err)
	}

	err = os.WriteFile(filename, bytes, 0644)
	if err != nil {
		return fmt.Errorf("Could not open file '%s' to write data: %v", filename, err)
	}

	return nil
}
