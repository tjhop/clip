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

package template

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// NB: nothing in this file may call t.Parallel(): t.Setenv() panics when used
// from a parallel test, and the sprout `env` cases rely on it.

func TestRender(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		tmplVars   map[string]string
		globalVars map[string]string
		// env is applied with t.Setenv, so it is scoped to the subtest
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name:       "global var is substituted",
			text:       `Hello, {{ .name }}!`,
			globalVars: map[string]string{"name": "x"},
			want:       "Hello, x!",
		},
		{
			name:       "template var overrides global var of the same name",
			text:       `Hello, {{ .name }}!`,
			globalVars: map[string]string{"name": "global"},
			tmplVars:   map[string]string{"name": "template"},
			want:       "Hello, template!",
		},
		{
			name:       "template var is substituted with no globals at all",
			text:       `Hello, {{ .name }}!`,
			globalVars: nil,
			tmplVars:   map[string]string{"name": "template"},
			want:       "Hello, template!",
		},
		{
			name: "undefined var renders as no value",
			text: `Hello, {{ .nope }}!`,
			want: "Hello, <no value>!",
		},
		{
			// sprout's `env` and `default`, per the README example
			name: "sprout functions are loaded, env var set",
			text: `Hello, {{ env "NAME" | default "Name unset" }}!`,
			env:  map[string]string{"NAME": "McLovin"},
			want: "Hello, McLovin!",
		},
		{
			name: "sprout functions are loaded, env var empty",
			text: `Hello, {{ env "NAME" | default "Name unset" }}!`,
			env:  map[string]string{"NAME": ""},
			want: "Hello, Name unset!",
		},
		{
			name:    "malformed template text returns a parse error",
			text:    `{{ .unclosed`,
			wantErr: true,
		},
		{
			name:    "reference to an undefined function returns a parse error",
			text:    `{{ definitelyNotASproutFunction "x" }}`,
			wantErr: true,
		},
		{
			name:    "reference to an undefined named template returns an execute error",
			text:    `{{ template "missing" }}`,
			wantErr: true,
		},
		{
			name:     "field access on a string var returns an execute error",
			text:     `{{ .name.foo }}`,
			tmplVars: map[string]string{"name": "x"},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Render reports every failure as an error; a panic is a bug.
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Render(%q) panicked: %v", tc.text, r)
				}
			}()

			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			var tmpl File
			tmpl.Template.Vars = tc.tmplVars
			tmpl.Template.Text = tc.text

			got, err := Render(tmpl, tc.globalVars)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("Render(%q) error = nil, want non-nil", tc.text)
				}
				if got != "" {
					t.Errorf("Render(%q) = %q, want empty string alongside error", tc.text, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("Render(%q) unexpected error: %v", tc.text, err)
			}
			if got != tc.want {
				t.Errorf("Render(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("loads tags, vars, and text", func(t *testing.T) {
		// the raw string literal intentionally starts at column 0 so that go
		// source indentation does not leak into the yaml
		const contents = `tags:
  - personal
  - demo
template:
  vars:
    value: Hello, world!
  text: |
    The value of the variable is: "{{ .value }}"
`

		// File.Template is an anonymous struct, so `want` has to be built by
		// assignment rather than as a composite literal
		var want File
		want.Tags = []string{"personal", "demo"}
		want.Template.Vars = map[string]string{"value": "Hello, world!"}
		// `text: |` is a literal block scalar, so the trailing newline is kept
		want.Template.Text = "The value of the variable is: \"{{ .value }}\"\n"

		filename := filepath.Join(t.TempDir(), "demo.yml")
		if err := os.WriteFile(filename, []byte(contents), 0o644); err != nil {
			t.Fatalf("failed to write test template file: %v", err)
		}

		got, err := Load(filename)
		if err != nil {
			t.Fatalf("Load(%q) unexpected error: %v", filename, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Load(%q) = %#v, want %#v", filename, got, want)
		}
	})

	t.Run("missing file returns an error", func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), "does-not-exist.yml")

		got, err := Load(filename)
		if err == nil {
			t.Fatalf("Load(%q) error = nil, want non-nil", filename)
		}
		if !reflect.DeepEqual(got, File{}) {
			t.Errorf("Load(%q) = %#v, want the zero value alongside error", filename, got)
		}
	})

	t.Run("malformed yaml returns an error", func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), "malformed.yml")
		if err := os.WriteFile(filename, []byte("tags: [unterminated\n"), 0o644); err != nil {
			t.Fatalf("failed to write test template file: %v", err)
		}

		got, err := Load(filename)
		if err == nil {
			t.Fatalf("Load(%q) error = nil, want non-nil", filename)
		}
		if !reflect.DeepEqual(got, File{}) {
			t.Errorf("Load(%q) = %#v, want the zero value alongside error", filename, got)
		}
	})
}
