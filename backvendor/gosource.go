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
	"encoding/json"
	"go/build"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/vcs"

	"github.com/release-engineering/backvendor/backvendor/glide"
)

type RepoRoot struct {
	vcs.RepoRoot

	Version string
}

var log = logging.MustGetLogger("backvendor")
var errorNoImportPathComment = errors.New("no import path comment")

// GoSource represents a filesystem tree containing Go source code.
type GoSource struct {
	// Path to the top-level package
	Path string

	// Package is any import path in this project
	Package string

	// repoRoots maps apparent import paths to actual repositories
	repoRoots map[string]*RepoRoot

	// excludes is a map of paths to ignore in this project
	excludes map[string]struct{}

	// usesGodep is true if Godeps/Godeps.json is present
	usesGodep bool
}

// FindExcludes returns a slice of paths which match the provided
// globs.
func FindExcludes(path string, globs []string) ([]string, error) {
	excludes := make([]string, 0)
	for _, glob := range globs {
		matches, err := filepath.Glob(filepath.Join(path, glob))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			excludes = append(excludes, match)
		}
	}
	return excludes, nil
}

// NewGoSource returns a *GoSource for the given path path. The paths
// in excludes will not be considered when matching against the
// upstream repository.
func NewGoSource(path string, excludes []string) (*GoSource, error) {
	excl := make(map[string]struct{})
	for _, e := range excludes {
		excl[e] = struct{}{}
	}

	src := &GoSource{
		Path:     path,
		excludes: excl,
	}

	// Always read Godeps.json because we need to know whether
	// godep is in use (if so, files are modified when vendored).
	err := loadGodepsConf(src)
	if err != nil {
		return nil, err
	}

	// Always read glide.yaml because we need to know if there are
	// replacement repositories.
	ok, err := loadGlideConf(src)
	if err != nil {
		return nil, err
	}

	if !ok && src.Package == "" {
		if importPath, err := findImportComment(src); err == nil {
			src.Package = importPath
		} else if importPath, ok := importPathFromFilepath(path); ok {
			src.Package = importPath
		}
	}

	return src, nil
}

// loadGodepsConf parses Godeps/Godeps.json to extract the package
// name.
func loadGodepsConf(src *GoSource) error {
	type godepsConf struct {
		ImportPath string
	}
	conf := filepath.Join(src.Path, "Godeps", "Godeps.json")
	if _, skip := src.excludes[conf]; skip {
		return nil
	}
	f, err := os.Open(conf)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	src.usesGodep = true
	dec := json.NewDecoder(f)
	var godeps godepsConf
	err = dec.Decode(&godeps)
	if err != nil {
		return err
	}

	src.Package = godeps.ImportPath
	log.Debugf("import path found from Godeps/Godeps.json: %s", src.Package)
	return nil
}

// loadGlideConf parses glide.yaml to extract the package name and the
// import path repository replacements. It returns true if it parsed
// successfully.
func loadGlideConf(src *GoSource) (bool, error) {
	conf := filepath.Join(src.Path, "glide.yaml")
	if _, skip := src.excludes[conf]; skip {
		return false, nil
	}

	glide, err := glide.LoadGlide(src.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "decoding %s", conf)
	}

	src.Package = glide.Package
	log.Debugf("import path found from glide.yaml: %s", src.Package)

	// if there is no vendor folder, the dependencies are flattened
	_, err = os.Stat(filepath.Join(src.Path, "vendor"))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, errors.Wrapf(err, "stat 'vendor' for %s", conf)
	}

	repoRoots := make(map[string]*RepoRoot)
	for _, imp := range glide.Imports {
		theVcs := vcs.ByCmd(vcsGit) // default to git
		if imp.Repo == "" {
			root, err := vcs.RepoRootForImportPath(imp.Name, false)
			if err != nil {
				log.Infof("Skipping %v, could not determine repo root: %v", imp.Name, err)
				continue
			}
			imp.Repo = root.Repo
			theVcs = root.VCS
		}

		repoRoots[imp.Name] = &RepoRoot{
			RepoRoot: vcs.RepoRoot{
				VCS:  theVcs,
				Repo: imp.Repo,
				Root: imp.Name,
			},
			Version: imp.Version,
		}
	}

	src.repoRoots = repoRoots
	return true, nil
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
		// Skip these special directories since "vendor" and
		// "_override" contain local copies of dependencies
		// and "testdata" includes data files only used for
		// testing that can be safely ignored.
		case "vendor", "testdata", "_override":
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
func (src GoSource) Project(importPath string) (*RepoRoot, error) {
	if importPath == "" {
		importPath = src.Package
		if importPath == "" {
			return nil, ErrorNeedImportPath
		}
	}

	repoRoot, err := vcs.RepoRootForImportPath(importPath, false)
	return &RepoRoot{RepoRoot: *repoRoot}, err
}

func (src GoSource) RepoRootForImportPath(importPath string) (*RepoRoot, error) {
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
			return &RepoRoot{RepoRoot: *r}, nil
		}
	}

	return &RepoRoot{RepoRoot: *r}, err
}
