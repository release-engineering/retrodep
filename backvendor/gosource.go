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
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/vcs"
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

func NewGoSource(path string) *GoSource {
	return &GoSource{
		Path: path,
	}
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
