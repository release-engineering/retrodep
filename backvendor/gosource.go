// Copyright (C) 2018 Tim Waugh
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package backvendor // import "github.com/release-engineering/backvendor/backvendor"

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/vcs"
	"gopkg.in/yaml.v2"
)

// GoSource represents a filesystem tree containing Go source code.
type GoSource struct {
	// Path to the top-level package
	Path string

	// Package is any import path in this project
	Package string

	// reporoots maps apparent import paths to actual repositories
	reporoots map[string]*vcs.RepoRoot
}

type Glide struct {
	Package string
	Import  []struct {
		Package string
		Repo    string `json:omitempty`
	}
}

func NewGoSource(path string) *GoSource {
	src := &GoSource{
		Path: path,
	}

	f, err := os.Open(filepath.Join(path, "glide.yaml"))
	if err != nil {
		return src
	}
	defer f.Close()

	// There is a glide.yaml so inspect it
	dec := yaml.NewDecoder(f)
	var glide Glide
	err = dec.Decode(&glide)
	if err != nil {
		return src
	}

	src.Package = glide.Package
	reporoots := make(map[string]*vcs.RepoRoot)
	for _, imp := range glide.Import {
		if imp.Repo == "" {
			continue
		}

		reporoots[imp.Package] = &vcs.RepoRoot{
			VCS:  vcsGit,
			Repo: imp.Repo,
			Root: imp.Package,
		}
	}

	src.reporoots = reporoots
	return src
}

// Vendor returns the path to the vendored source code.
func (src GoSource) Vendor() string {
	return filepath.Join(src.Path, "vendor")
}

func (src GoSource) RepoRootForImportPath(importPath string) (*vcs.RepoRoot, error) {
	// First look up replacements
	path := importPath
	for {
		repl, ok := src.reporoots[path]
		if ok {
			// Found a replacement repo
			return repl, nil
		}

		slash := strings.LastIndex(path, "/")
		if slash < 1 {
			break
		}

		// Try shorter import path
		path = path[:slash]
	}

	// No replacement found, use the import path as-is
	return vcs.RepoRootForImportPath(importPath, false)
}
