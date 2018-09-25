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
	"go/build"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/vcs"
	"gopkg.in/yaml.v2"
)

var log = logging.MustGetLogger("backvendor")
var errorNoImportPathComment = errors.New("no import path comment")

// GoSource represents a filesystem tree containing Go source code.
type GoSource struct {
	// Path to the top-level package
	Path string

	// Package is any import path in this project
	Package string

	// repoRoots maps apparent import paths to actual repositories
	repoRoots map[string]*vcs.RepoRoot

	// excludes is a map of paths to ignore in this project
	excludes map[string]struct{}

	// usesGodep is true if Godeps/Godeps.json is present
	usesGodep bool
}

type glideConf struct {
	Package string
	Import  []struct {
		Package string
		Repo    string `json:"omitempty"`
	}
}

func findExcludes(pth string, globs []string) (map[string]struct{}, error) {
	excludes := make(map[string]struct{})
	for _, glob := range globs {
		matches, err := filepath.Glob(filepath.Join(pth, glob))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			excludes[match] = struct{}{}
		}
	}
	return excludes, nil
}

func NewGoSource(pth string, excludeGlobs ...string) (*GoSource, error) {
	excludes, err := findExcludes(pth, excludeGlobs)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(filepath.Join(pth, "Godeps", "Godeps.json"))
	src := &GoSource{
		Path:      pth,
		excludes:  excludes,
		usesGodep: err == nil,
	}

	if !readGlideConf(src) {
		if importPath, err := findImportComment(src); err == nil {
			src.Package = importPath
		} else if importPath, ok := importPathFromFilepath(pth); ok {
			src.Package = importPath
		}
	}

	return src, nil
}

// readGlideConf parses glide.yaml to extract the package name and the
// import path repository replacements. It returns true if it parsed
// successfully.
func readGlideConf(src *GoSource) bool {
	conf := filepath.Join(src.Path, "glide.yaml")
	if _, skip := src.excludes[conf]; skip {
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
	absPath, err := filepath.Abs(pth)
	if err != nil {
		return "", false
	}

	// Skip leading '/'
	pth = absPath[1:]
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
	// Define the error we'll use to end the filepath.Walk method early.
	errFound := errors.New("found")

	// importPath holds the import path we've discovered. It will
	// be updated by the 'search' closure, below.
	var importPath string

	search := func(pth string, info os.FileInfo, err error) error {
		if _, skip := src.excludes[pth]; skip {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if info.Name() != "." &&
			strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		switch info.Name() {
		// Skip these special directories since "vendor"
		// contains local copies of dependencies and
		// "testdata" includes data files only used for
		// testing that can be safely ignored.
		case "vendor", "testdata":
			return filepath.SkipDir
		}

		pkg, err := build.ImportDir(pth, build.ImportComment)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				return nil
			}
			return err
		}
		if pkg.ImportComment != "" {
			importPath = pkg.ImportComment
			log.Debugf("found import path from import comment: %s",
				importPath)
			return errFound
		}
		return nil
	}

	err := filepath.Walk(src.Path, search)
	if err == errFound {
		err = nil
	} else if err == nil {
		err = errorNoImportPathComment
	}
	return importPath, err
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
	pth := importPath
	for {
		repl, ok := src.repoRoots[pth]
		if ok {
			// Found a replacement repo
			return repl, nil
		}

		// Try shorter import path
		pth = path.Dir(pth)
		if len(pth) == 1 {
			break
		}
	}

	// No replacement found, use the import pth as-is
	r, err := vcs.RepoRootForImportPath(importPath, false)
	if err != nil && strings.ContainsRune(importPath, '_') {
		// gopkg.in gives bad responses for paths like
		// gopkg.in/foo/bar.v2/_examples/chat1
		// because of the underscore. Remove it and try again.
		u := strings.Index(importPath, "_")
		importPath = path.Dir(importPath[:u])
		r, nerr := vcs.RepoRootForImportPath(importPath, false)
		if nerr == nil {
			return r, nil
		}
	}

	return r, err
}
