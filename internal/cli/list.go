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
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"

	"github.com/tjhop/clip/internal/config"
	"github.com/tjhop/clip/internal/template"
)

var (
	listCmdName  string
	listTags     []string
	listTagsOnly bool
)

func registerListCmd(app *kingpin.Application) {
	cmd := app.Command("list", "List available Clip templates/tags (default if just running `clip`).")
	cmd.Default()
	cmd.HelpLong(`List available Clip templates (can be filtered by tag) or list the available tags.

Example:
  clip list
  clip list --tags-only
  clip list --tags personal,work`)
	cmd.Flag("tags", "Comma separated list of tags to filter templates.").StringsVar(&listTags)
	cmd.Flag("tags-only", "List all tags used in the templates.").BoolVar(&listTagsOnly)
	cmd.Flag("list-tags", "Alias for '--tags-only' flag.").Hidden().BoolVar(&listTagsOnly)
	cmd.Flag("show-tags", "Alias for '--tags-only' flag.").Hidden().BoolVar(&listTagsOnly)
	listCmdName = cmd.FullCommand()
}

func runList(cfg config.Config) error {
	lines, err := listLines(cfg.TemplateDir, splitTags(listTags), listTagsOnly)
	if err != nil {
		return err
	}

	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}

// splitTags expands comma separated values so both `--tags a,b` and
// `--tags a --tags b` work.
func splitTags(values []string) []string {
	var tags []string
	for _, v := range values {
		for _, tag := range strings.Split(v, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

// listLines walks dir in lexical order and returns what `clip list` prints:
// template names (path relative to dir, extension stripped, which is what the
// other commands take), or with tagsOnly the sorted set of tags in use. A
// non-empty tags filter keeps only templates carrying one of them. Template
// files are only parsed when tags are involved, so a malformed template
// doesn't break a plain `clip list`.
func listLines(dir string, tags []string, tagsOnly bool) ([]string, error) {
	var names []string
	seen := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		if d.IsDir() || !slices.Contains(templateExts, ext) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(strings.TrimSuffix(rel, ext))

		if !tagsOnly && len(tags) == 0 {
			names = append(names, name)
			return nil
		}
		tmpl, err := template.Load(path)
		if err != nil {
			return fmt.Errorf("could not load Clip template '%s': %w", name, err)
		}
		for _, tag := range tmpl.Tags {
			seen[tag] = true
		}
		if slices.ContainsFunc(tags, func(tag string) bool { return slices.Contains(tmpl.Tags, tag) }) {
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not walk template directory '%s': %w", dir, err)
	}

	if tagsOnly {
		return slices.Sorted(maps.Keys(seen)), nil
	}
	return names, nil
}
