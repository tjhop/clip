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

package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// testHome stands in for $HOME. Expected values are literals, not derived from
// the package constants, so the table checks the rules rather than the
// implementation against itself.
const testHome = "/home/test"

var (
	// wantDefaultDir is testHome's built-in template directory.
	wantDefaultDir = filepath.Join(testHome, "clip")

	// wantDefaultVars is the built-in global template vars.
	wantDefaultVars = map[string]string{"name": "Clip User"}
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		env   map[string]string
		file  Config
		want  Config
	}{
		// editor: flag > $EDITOR > file > nano
		{
			name: "editor unset everywhere falls back to default",
			want: Config{Editor: "nano", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name: "editor from file only",
			file: Config{Editor: "emacs"},
			want: Config{Editor: "emacs", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name: "editor from env only",
			env:  map[string]string{"EDITOR": "vim"},
			want: Config{Editor: "vim", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name: "editor env beats file",
			env:  map[string]string{"EDITOR": "vim"},
			file: Config{Editor: "emacs"},
			want: Config{Editor: "vim", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name:  "editor from flag only",
			flags: Flags{Editor: "helix"},
			want:  Config{Editor: "helix", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name:  "editor flag beats file",
			flags: Flags{Editor: "helix"},
			file:  Config{Editor: "emacs"},
			want:  Config{Editor: "helix", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name:  "editor flag beats env",
			flags: Flags{Editor: "helix"},
			env:   map[string]string{"EDITOR": "vim"},
			want:  Config{Editor: "helix", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name:  "editor flag beats env and file",
			flags: Flags{Editor: "helix"},
			env:   map[string]string{"EDITOR": "vim"},
			file:  Config{Editor: "emacs"},
			want:  Config{Editor: "helix", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name: "empty env value falls through to file",
			env:  map[string]string{"EDITOR": ""},
			file: Config{Editor: "emacs"},
			want: Config{Editor: "emacs", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},

		// templatedir: flag > file > <home>/clip
		{
			name: "templatedir unset everywhere falls back to home",
			want: Config{Editor: "nano", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			// Regression guard: a config file's templatedir must beat the
			// built-in default.
			name: "templatedir from config file wins over built-in default (regression)",
			file: Config{TemplateDir: "/custom"},
			want: Config{Editor: "nano", TemplateDir: "/custom", Vars: wantDefaultVars},
		},
		{
			name: "templatedir ~/ in config file expands to home",
			file: Config{TemplateDir: "~/tpl"},
			want: Config{Editor: "nano", TemplateDir: filepath.Join(testHome, "tpl"), Vars: wantDefaultVars},
		},
		{
			name: "templatedir bare ~ in config file expands to home",
			file: Config{TemplateDir: "~"},
			want: Config{Editor: "nano", TemplateDir: testHome, Vars: wantDefaultVars},
		},
		{
			name:  "templatedir from flag only",
			flags: Flags{TemplateDir: "/flagdir"},
			want:  Config{Editor: "nano", TemplateDir: "/flagdir", Vars: wantDefaultVars},
		},
		{
			name:  "templatedir flag beats file",
			flags: Flags{TemplateDir: "/flagdir"},
			file:  Config{TemplateDir: "/custom"},
			want:  Config{Editor: "nano", TemplateDir: "/flagdir", Vars: wantDefaultVars},
		},
		{
			name:  "templatedir ~ in flag is left to the shell",
			flags: Flags{TemplateDir: "~/flagdir"},
			want:  Config{Editor: "nano", TemplateDir: "~/flagdir", Vars: wantDefaultVars},
		},

		// vars: file only, defaulted when the key is absent
		{
			name: "nil vars in file fall back to default vars",
			file: Config{Vars: nil},
			want: Config{Editor: "nano", TemplateDir: wantDefaultDir, Vars: wantDefaultVars},
		},
		{
			name: "empty vars in file are kept empty",
			file: Config{Vars: map[string]string{}},
			want: Config{Editor: "nano", TemplateDir: wantDefaultDir, Vars: map[string]string{}},
		},
		{
			name: "vars from file replace default vars",
			file: Config{Vars: map[string]string{"name": "captain", "org": "clip"}},
			want: Config{Editor: "nano", TemplateDir: wantDefaultDir, Vars: map[string]string{"name": "captain", "org": "clip"}},
		},

		// everything at once
		{
			name:  "all settings from all sources",
			flags: Flags{Editor: "helix", TemplateDir: "/flagdir"},
			env:   map[string]string{"EDITOR": "vim"},
			file:  Config{Editor: "emacs", TemplateDir: "/custom", Vars: map[string]string{"name": "captain"}},
			want:  Config{Editor: "helix", TemplateDir: "/flagdir", Vars: map[string]string{"name": "captain"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }

			// reflect.DeepEqual distinguishes a nil map from an empty one, which
			// is exactly the vars distinction under test.
			if got := Resolve(tc.flags, getenv, tc.file, testHome); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Resolve() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestResolveOnlyReadsEditorFromEnv(t *testing.T) {
	var read []string
	getenv := func(key string) string {
		read = append(read, key)
		return ""
	}

	Resolve(Flags{}, getenv, Config{}, testHome)

	if want := []string{"EDITOR"}; !reflect.DeepEqual(read, want) {
		t.Errorf("Resolve() read env vars %v, want %v", read, want)
	}
}

func TestDefaultPath(t *testing.T) {
	if got, want := DefaultPath(testHome), filepath.Join(testHome, ".clip.yml"); got != want {
		t.Errorf("DefaultPath(%q) = %q, want %q", testHome, got, want)
	}
}

func TestLoad(t *testing.T) {
	write := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), ".clip.yml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test config file: %v", err)
		}
		return path
	}

	t.Run("all keys present", func(t *testing.T) {
		path := write(t, "editor: vim\ntemplatedir: /custom\nvars:\n  name: captain\n  org: clip\n")

		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}

		want := Config{
			Editor:      "vim",
			TemplateDir: "/custom",
			Vars:        map[string]string{"name": "captain", "org": "clip"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Load() = %#v, want %#v", got, want)
		}
	})

	t.Run("empty file loads the zero config", func(t *testing.T) {
		got, err := Load(write(t, ""))
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if !reflect.DeepEqual(got, Config{}) {
			t.Errorf("Load() = %#v, want zero Config", got)
		}
	})

	t.Run("null vars load as a nil map", func(t *testing.T) {
		got, err := Load(write(t, "vars:\n"))
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if got.Vars != nil {
			t.Errorf("Load().Vars = %#v, want nil", got.Vars)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "nope.yml")); err == nil {
			t.Error("Load() error = nil, want error for missing file")
		}
	})

	t.Run("malformed yaml returns error", func(t *testing.T) {
		if _, err := Load(write(t, "editor: [unterminated\n")); err == nil {
			t.Error("Load() error = nil, want error for malformed yaml")
		}
	})
}

func TestEnsureConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".clip.yml")

	created, err := EnsureConfigFile(path)
	if err != nil {
		t.Fatalf("EnsureConfigFile() error = %v, want nil", err)
	}
	if !created {
		t.Error("EnsureConfigFile() created = false, want true for a missing file")
	}

	// The scaffold must parse, pin only editor and vars, and mention every key
	// (even commented out) so users can find them in the file.
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() of the scaffold error = %v, want nil", err)
	}
	want := Config{Editor: "nano", Vars: map[string]string{"name": "Clip User"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() of the scaffold = %#v, want %#v", got, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written config file: %v", err)
	}
	for _, key := range []string{"editor:", "templatedir:", "vars:"} {
		if !strings.Contains(string(data), key) {
			t.Errorf("scaffold does not mention the %q key", key)
		}
	}

	// An existing file is left alone, whatever it contains.
	sentinel := []byte("editor: vim\n")
	if err := os.WriteFile(path, sentinel, 0o644); err != nil {
		t.Fatalf("failed to overwrite config file: %v", err)
	}

	created, err = EnsureConfigFile(path)
	if err != nil {
		t.Fatalf("EnsureConfigFile() error = %v, want nil", err)
	}
	if created {
		t.Error("EnsureConfigFile() created = true, want false for an existing file")
	}

	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to re-read config file: %v", err)
	}
	if !bytes.Equal(data, sentinel) {
		t.Errorf("existing config file was rewritten: got %q, want %q", data, sentinel)
	}
}

func TestEnsureConfigFileMissingParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", ".clip.yml")

	created, err := EnsureConfigFile(path)
	if err == nil {
		t.Error("EnsureConfigFile() error = nil, want error for a missing parent directory")
	}
	if created {
		t.Error("EnsureConfigFile() created = true, want false on error")
	}
}

func TestEnsureTemplateDir(t *testing.T) {
	// Nest the target so MkdirAll's parent creation is exercised too.
	path := filepath.Join(t.TempDir(), "nested", "clip")

	created, err := EnsureTemplateDir(path)
	if err != nil {
		t.Fatalf("EnsureTemplateDir() error = %v, want nil", err)
	}
	if !created {
		t.Error("EnsureTemplateDir() created = false, want true for a missing directory")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat created template dir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("EnsureTemplateDir() created %s, want a directory", path)
	}

	created, err = EnsureTemplateDir(path)
	if err != nil {
		t.Fatalf("EnsureTemplateDir() error = %v, want nil", err)
	}
	if created {
		t.Error("EnsureTemplateDir() created = true, want false for an existing directory")
	}
}
