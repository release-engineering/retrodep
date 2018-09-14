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
	"bufio"
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

	// repoRoots maps apparent import paths to actual repositories
	repoRoots map[string]*vcs.RepoRoot

	// excludes is a map of paths to ignore in this project
	excludes map[string]bool
}

type glideConf struct {
	Package string
	Import  []struct {
		Package string
		Repo    string `json:omitempty`
	}
}

func findExcludes(path string, globs []string) map[string]bool {
	excludes := make(map[string]bool)
	for _, glob := range globs {
		matches, err := filepath.Glob(filepath.Join(path, glob))
		if err != nil {
			continue
		}
		for _, match := range matches {
			excludes[match] = true
		}
	}
	return excludes
}

func NewGoSource(path string, excludeGlobs ...string) *GoSource {
	src := &GoSource{
		Path:     path,
		excludes: findExcludes(path, excludeGlobs),
	}

	if !readGlideConf(src) {
		if importPath, err := findImportComment(src); err == nil {
			src.Package = importPath
		} else if importPath, ok := importPathFromFilepath(path); ok {
			src.Package = importPath
		}
	}

	return src
}

// readGlideConf parses glide.yaml to extract the package name and the
// import path repository replacements. It returns true if it parsed
// successfully.
func readGlideConf(src *GoSource) bool {
	conf := filepath.Join(src.Path, "glide.yaml")
	if src.excludes[conf] {
		return false
	}
	f, err := os.Open(conf)
	if err != nil {
		return false
	}
	defer f.Close()

	// There is a glide.yaml so inspect it
	dec := yaml.NewDecoder(f)
	var glide glideConf
	err = dec.Decode(&glide)
	if err != nil {
		return false
	}

	src.Package = glide.Package
	repoRoots := make(map[string]*vcs.RepoRoot)
	for _, imp := range glide.Import {
		if imp.Repo == "" {
			continue
		}

		repoRoots[imp.Package] = &vcs.RepoRoot{
			VCS:  vcs.ByCmd(vcsGit),
			Repo: imp.Repo,
			Root: imp.Package,
		}
	}

	src.repoRoots = repoRoots
	return true
}

// importPathFromFilepath attempts to use the project directory path to
// infer its import path.
func importPathFromFilepath(pth string) (string, bool) {
	if len(pth) < 1 {
		return "", false
	}

	var err error
	if pth == "." {
		pth, err = os.Getwd()
		if err != nil {
			return "", false
		}
	}

	if pth[0] == filepath.Separator {
		pth = pth[1:]
	}

	components := strings.Split(pth, string(filepath.Separator))
	if len(components) < 2 {
		return "", false
	}

	for i := len(components) - 2; i >= 0; i -= 1 {
		if strings.Index(components[i], ".") == -1 {
			// Not a hostname
			continue
		}

		pth := strings.Join(components[i:len(components)], "/")
		repoRoot, err := vcs.RepoRootForImportPath(pth, false)
		if err == nil {
			return repoRoot.Root, true
		}
	}

	return "", false
}

func findImportComment(src *GoSource) (string, error) {
	var importPath string
	search := func(path string, info os.FileInfo, err error) error {
		if src.excludes[path] || importPath != "" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() != "." &&
				strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			if info.Name() == "vendor" || info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(bufio.NewReader(r))
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			if fields[0] == "//" || fields[0] == "/*" {
				continue
			}
			if fields[0] != "package" {
				return nil
			}
			if fields[2] != "/*" && fields[2] != "//" {
				return nil
			}
			if fields[3] != "import" {
				return nil
			}
			path := fields[4]
			if len(path) < 3 {
				return nil
			}
			if path[0] != '"' ||
				path[len(path)-1] != '"' {
				return nil
			}
			importPath = path[1 : len(path)-1]
			break
		}
		return nil
	}

	err := filepath.Walk(src.Path, search)
	if err != nil {
		return "", err
	}
	return importPath, nil
}

// Vendor returns the path to the vendored source code.
func (src GoSource) Vendor() string {
	return filepath.Join(src.Path, "vendor")
}

// Project returns information about the project given its import
// path. If importPath is "" it is deduced from import comments, if
// available.
func (src GoSource) Project(importPath string) (*vcs.RepoRoot, error) {
	if importPath == "" {
		importPath = src.Package
		if importPath == "" {
			return nil, ErrorNeedImportPath
		}
	}

	repoRoot, err := vcs.RepoRootForImportPath(importPath, false)
	return repoRoot, err
}

func (src GoSource) RepoRootForImportPath(importPath string) (*vcs.RepoRoot, error) {
	// First look up replacements
	path := importPath
	for {
		repl, ok := src.repoRoots[path]
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
