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

package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/tjhop/clip/internal/config"
)

// NB: nothing in this file may call t.Parallel(): kingpin writes parsed values
// into package-level variables, and the config tests use t.Setenv.

// resetFlags zeroes every flag and argument target. kingpin never clears them,
// so each parse in a test must start from a clean slate.
func resetFlags() {
	flagConfigFile, flagTemplateDir = "", ""
	copyTemplate = ""
	createTemplate = ""
	editTemplate, editEditor = "", ""
	listTags, listTagsOnly = nil, false
	removeTemplate = ""
	renameOld, renameNew = "", ""
	showTemplate = ""
}

func TestCompletionArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, nil},
		{"ordinary invocation is unchanged", []string{"copy", "foo"}, []string{"copy", "foo"}},
		{"request with no word is unchanged", []string{"--completion-bash"}, []string{"--completion-bash"}},
		{"partial top-level word is dropped", []string{"--completion-bash", "li"}, []string{"--completion-bash"}},
		{"empty top-level word is dropped", []string{"--completion-bash", ""}, []string{"--completion-bash"}},
		{"partial argument word is dropped", []string{"--completion-bash", "copy", "fo"}, []string{"--completion-bash", "copy"}},
		{"partial word after an app flag is dropped", []string{"--completion-bash", "-t", "dir", "sh"}, []string{"--completion-bash", "-t", "dir"}},
		{"partial flag is kept", []string{"--completion-bash", "--te"}, []string{"--completion-bash", "--te"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := slices.Clone(tc.in)

			got := CompletionArgs(in)
			if !slices.Equal(got, tc.want) {
				t.Errorf("CompletionArgs(%v) = %v, want %v", tc.in, got, tc.want)
			}
			if !slices.Equal(in, tc.in) {
				t.Errorf("CompletionArgs modified its input: %v, want %v", in, tc.in)
			}
		})
	}
}

// parsed captures the parse outcome TestParse compares.
type parsed struct {
	cmd          string
	templateDir  string
	copyTemplate string
	renameOld    string
	renameNew    string
	tags         []string
}

// TestParse runs arguments through kingpin itself and pins what main relies
// on: list as the default command, list's flags accepted before any command,
// an unknown first word rejected rather than taken as a template name, and
// the arguments each command receives.
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    parsed
		wantErr bool
	}{
		{"bare invocation lists", nil, parsed{cmd: "list"}, false},
		{"explicit copy", []string{"copy", "foo"}, parsed{cmd: "copy", copyTemplate: "foo"}, false},
		{"alias resolves to the command", []string{"load", "foo"}, parsed{cmd: "copy", copyTemplate: "foo"}, false},
		{"unknown first word is an error", []string{"foo"}, parsed{}, true},
		{"flag terminator does not make a template name", []string{"--", "foo"}, parsed{}, true},
		{"empty argument is an error", []string{""}, parsed{}, true},
		{"subcommand flag before any command falls back to list", []string{"--tags", "a,b"}, parsed{cmd: "list", tags: []string{"a,b"}}, false},
		{"attached short flag value", []string{"-tdir", "list"}, parsed{cmd: "list", templateDir: "dir"}, false},
		{"separate short flag value", []string{"-t", "dir", "copy", "foo"}, parsed{cmd: "copy", templateDir: "dir", copyTemplate: "foo"}, false},
		{"compat flag spelling", []string{"--templatedir", "dir", "list"}, parsed{cmd: "list", templateDir: "dir"}, false},
		{"rename takes two args", []string{"rename", "a", "b"}, parsed{cmd: "rename", renameOld: "a", renameNew: "b"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			app := NewApp()

			cmd, err := app.Parse(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%v) error = nil, want error", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%v) error = %v, want nil", tc.args, err)
			}

			got := parsed{cmd, flagTemplateDir, copyTemplate, renameOld, renameNew, listTags}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%v) = %+v, want %+v", tc.args, got, tc.want)
			}
		})
	}
}

func TestSplitTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, nil},
		{"comma separated", []string{"a,b"}, []string{"a", "b"}},
		{"repeated flags", []string{"a", "b"}, []string{"a", "b"}},
		{"mixed with spaces and empties", []string{" a ,", "b,c"}, []string{"a", "b", "c"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := splitTags(tc.in); !slices.Equal(got, tc.want) {
				t.Errorf("splitTags(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// writeFile writes content to the path rel under dir, creating parents.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// readFile returns the content of the path rel under dir, or "" if absent.
func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatalf("failed to read %s: %v", rel, err)
	}
	return string(data)
}

func TestListLines(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bar.yml", "tags: [work, golang]\ntemplate:\n  text: hi\n")
	writeFile(t, dir, "foo.yaml", "tags: [personal]\ntemplate:\n  text: hi\n")
	writeFile(t, dir, "sub/baz.yml", "tags: [work]\ntemplate:\n  text: hi\n")
	writeFile(t, dir, "notes.txt", "not a template")

	tests := []struct {
		name     string
		tags     []string
		tagsOnly bool
		want     []string
	}{
		{"no filter lists every template by its name", nil, false, []string{"bar", "foo", "sub/baz"}},
		{"tag filter", []string{"work"}, false, []string{"bar", "sub/baz"}},
		{"tag filter matches whole tags only", []string{"go"}, false, nil},
		{"any of several tags", []string{"personal", "golang"}, false, []string{"bar", "foo"}},
		{"no match", []string{"nothing"}, false, nil},
		{"tags only are deduplicated and sorted", nil, true, []string{"golang", "personal", "work"}},
		{"tags only ignores the tag filter", []string{"work"}, true, []string{"golang", "personal", "work"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := listLines(dir, tc.tags, tc.tagsOnly)
			if err != nil {
				t.Fatalf("listLines() error = %v, want nil", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("listLines(tags=%v, tagsOnly=%v) = %v, want %v", tc.tags, tc.tagsOnly, got, tc.want)
			}
		})
	}

	t.Run("missing directory is an error", func(t *testing.T) {
		if _, err := listLines(filepath.Join(dir, "missing"), nil, false); err == nil {
			t.Error("listLines() error = nil, want error for a missing directory")
		}
	})

	t.Run("a malformed template only fails when its tags are read", func(t *testing.T) {
		writeFile(t, dir, "broken.yml", "tags: [\n")
		if _, err := listLines(dir, nil, false); err != nil {
			t.Errorf("listLines() without tags error = %v, want nil", err)
		}
		if _, err := listLines(dir, []string{"work"}, false); err == nil {
			t.Error("listLines() with a tag filter error = nil, want a parse error")
		}
		if _, err := listLines(dir, nil, true); err == nil {
			t.Error("listLines() with tagsOnly error = nil, want a parse error")
		}
	})
}

func TestHandlers(t *testing.T) {
	t.Run("create writes the scaffold and refuses to overwrite", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir()}
		createTemplate = "foo"

		if err := runCreate(cfg); err != nil {
			t.Fatalf("runCreate() error = %v, want nil", err)
		}
		if got := readFile(t, cfg.TemplateDir, "foo.yml"); got != templateScaffold {
			t.Errorf("created template = %q, want the scaffold", got)
		}

		writeFile(t, cfg.TemplateDir, "foo.yml", "custom")
		if err := runCreate(cfg); !errors.Is(err, fs.ErrExist) {
			t.Errorf("runCreate() on an existing template error = %v, want fs.ErrExist", err)
		}
		if got := readFile(t, cfg.TemplateDir, "foo.yml"); got != "custom" {
			t.Errorf("existing template was overwritten: %q", got)
		}
	})

	t.Run("create makes parent directories for nested names", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir()}
		createTemplate = "sub/foo"

		if err := runCreate(cfg); err != nil {
			t.Fatalf("runCreate() error = %v, want nil", err)
		}
		if got := readFile(t, cfg.TemplateDir, "sub/foo.yml"); got != templateScaffold {
			t.Errorf("nested template = %q, want the scaffold", got)
		}
	})

	t.Run("names that escape the template directory are rejected", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir()}

		for _, name := range []string{"", "../escape", "/abs", "sub/../../escape"} {
			createTemplate, showTemplate = name, name
			if err := runCreate(cfg); err == nil {
				t.Errorf("runCreate(%q) error = nil, want error", name)
			}
			if err := runShow(cfg); err == nil {
				t.Errorf("runShow(%q) error = nil, want error", name)
			}
		}
	})

	t.Run("show and remove report a missing template", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir()}
		writeFile(t, cfg.TemplateDir, "foo.yml", "tags: []\n")

		showTemplate = "foo"
		if err := runShow(cfg); err != nil {
			t.Errorf("runShow() error = %v, want nil", err)
		}
		showTemplate = "nope"
		if err := runShow(cfg); !errors.Is(err, errTemplateNotFound) {
			t.Errorf("runShow() on a missing template error = %v, want errTemplateNotFound", err)
		}

		removeTemplate = "nope"
		if err := runRemove(cfg); !errors.Is(err, errTemplateNotFound) {
			t.Errorf("runRemove() on a missing template error = %v, want errTemplateNotFound", err)
		}
		removeTemplate = "foo"
		if err := runRemove(cfg); err != nil {
			t.Errorf("runRemove() error = %v, want nil", err)
		}
		if got := readFile(t, cfg.TemplateDir, "foo.yml"); got != "" {
			t.Errorf("removed template still exists with content %q", got)
		}
	})

	t.Run("rename moves a template, keeps its extension, and refuses to clobber", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir()}
		writeFile(t, cfg.TemplateDir, "a.yaml", "a")
		writeFile(t, cfg.TemplateDir, "c.yml", "c")

		renameOld, renameNew = "a", "b"
		if err := runRename(cfg); err != nil {
			t.Fatalf("runRename() error = %v, want nil", err)
		}
		if got := readFile(t, cfg.TemplateDir, "a.yaml"); got != "" {
			t.Errorf("source template still exists with content %q", got)
		}
		if got := readFile(t, cfg.TemplateDir, "b.yaml"); got != "a" {
			t.Errorf("renamed template = %q, want %q", got, "a")
		}

		renameOld, renameNew = "b", "c"
		if err := runRename(cfg); err == nil {
			t.Error("runRename() onto an existing template error = nil, want error")
		}
		if got := readFile(t, cfg.TemplateDir, "b.yaml"); got != "a" {
			t.Errorf("source template changed on refused rename: %q", got)
		}
		if got := readFile(t, cfg.TemplateDir, "c.yml"); got != "c" {
			t.Errorf("destination template changed on refused rename: %q", got)
		}

		renameOld, renameNew = "nope", "d"
		if err := runRename(cfg); !errors.Is(err, errTemplateNotFound) {
			t.Errorf("runRename() of a missing template error = %v, want errTemplateNotFound", err)
		}
		if got := readFile(t, cfg.TemplateDir, "d.yml"); got != "" {
			t.Errorf("destination was created for a missing source: %q", got)
		}
	})

	t.Run("edit scaffolds a missing template and passes editor arguments through", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir(), Editor: "true --flag"}
		editTemplate = "foo"

		if err := runEdit(cfg); err != nil {
			t.Fatalf("runEdit() error = %v, want nil", err)
		}
		if got := readFile(t, cfg.TemplateDir, "foo.yml"); got != templateScaffold {
			t.Errorf("edited template = %q, want the scaffold", got)
		}
	})

	t.Run("edit with an unknown editor fails before scaffolding", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir(), Editor: "clip-test-no-such-editor"}
		editTemplate = "foo"

		if err := runEdit(cfg); err == nil {
			t.Error("runEdit() error = nil, want error for an unknown editor")
		}
		if got := readFile(t, cfg.TemplateDir, "foo.yml"); got != "" {
			t.Errorf("template was scaffolded despite the editor failing: %q", got)
		}
	})

	t.Run("copy of a missing template fails before touching the clipboard", func(t *testing.T) {
		resetFlags()
		cfg := config.Config{TemplateDir: t.TempDir()}
		copyTemplate = "nope"

		if err := runCopy(cfg); !errors.Is(err, errTemplateNotFound) {
			t.Errorf("runCopy() error = %v, want errTemplateNotFound", err)
		}
	})
}

func TestResolveConfig(t *testing.T) {
	// setup gives each case its own HOME, no $EDITOR, and clean flags.
	setup := func(t *testing.T) string {
		t.Helper()
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("EDITOR", "")
		resetFlags()
		t.Cleanup(resetFlags)
		return home
	}

	t.Run("first run creates the config file and template directory", func(t *testing.T) {
		home := setup(t)

		cfg, err := resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}

		want := config.Config{Editor: "nano", TemplateDir: filepath.Join(home, "clip"), Vars: map[string]string{"name": "Clip User"}}
		if !reflect.DeepEqual(cfg, want) {
			t.Errorf("resolveConfig() = %#v, want %#v", cfg, want)
		}
		if got := readFile(t, home, ".clip.yml"); got == "" {
			t.Error("config file was not created")
		}
		if info, err := os.Stat(cfg.TemplateDir); err != nil || !info.IsDir() {
			t.Errorf("template directory was not created: %v", err)
		}
	})

	t.Run("config file templatedir wins over the default and is created", func(t *testing.T) {
		home := setup(t)
		custom := filepath.Join(home, "custom")
		writeFile(t, home, ".clip.yml", "templatedir: "+custom+"\n")

		cfg, err := resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if cfg.TemplateDir != custom {
			t.Errorf("resolveConfig().TemplateDir = %q, want %q", cfg.TemplateDir, custom)
		}
		if info, err := os.Stat(custom); err != nil || !info.IsDir() {
			t.Errorf("configured template directory was not created: %v", err)
		}
	})

	t.Run("tilde in config file templatedir expands to home", func(t *testing.T) {
		home := setup(t)
		writeFile(t, home, ".clip.yml", "templatedir: ~/tpl\n")

		cfg, err := resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if want := filepath.Join(home, "tpl"); cfg.TemplateDir != want {
			t.Errorf("resolveConfig().TemplateDir = %q, want %q", cfg.TemplateDir, want)
		}
	})

	t.Run("template-dir flag beats the config file", func(t *testing.T) {
		home := setup(t)
		writeFile(t, home, ".clip.yml", "templatedir: "+filepath.Join(home, "custom")+"\n")
		flagTemplateDir = filepath.Join(home, "flagdir")

		cfg, err := resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if cfg.TemplateDir != flagTemplateDir {
			t.Errorf("resolveConfig().TemplateDir = %q, want %q", cfg.TemplateDir, flagTemplateDir)
		}
	})

	t.Run("EDITOR beats the config file and the editor flag beats EDITOR", func(t *testing.T) {
		home := setup(t)
		writeFile(t, home, ".clip.yml", "editor: emacs\n")
		t.Setenv("EDITOR", "vim")

		cfg, err := resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if cfg.Editor != "vim" {
			t.Errorf("resolveConfig().Editor = %q, want %q", cfg.Editor, "vim")
		}

		editEditor = "helix"
		cfg, err = resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if cfg.Editor != "helix" {
			t.Errorf("resolveConfig().Editor = %q, want %q", cfg.Editor, "helix")
		}
	})

	t.Run("config flag path is created and then read", func(t *testing.T) {
		home := setup(t)
		flagConfigFile = filepath.Join(home, "alt.yml")

		if _, err := resolveConfig(); err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if got := readFile(t, home, "alt.yml"); got == "" {
			t.Fatal("alternate config file was not created")
		}
		if got := readFile(t, home, ".clip.yml"); got != "" {
			t.Error("default config file was created despite --config")
		}

		writeFile(t, home, "alt.yml", "editor: emacs\n")
		cfg, err := resolveConfig()
		if err != nil {
			t.Fatalf("resolveConfig() error = %v, want nil", err)
		}
		if cfg.Editor != "emacs" {
			t.Errorf("resolveConfig().Editor = %q, want %q", cfg.Editor, "emacs")
		}
	})

	t.Run("malformed config file is an error", func(t *testing.T) {
		home := setup(t)
		writeFile(t, home, ".clip.yml", "editor: [unterminated\n")

		if _, err := resolveConfig(); err == nil {
			t.Error("resolveConfig() error = nil, want error for a malformed config file")
		}
	})
}
