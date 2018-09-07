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

package backvendor

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/vcs"
)

func pathStartsWith(dir, prefix string) bool {
	return strings.HasPrefix(dir, prefix) &&
		(len(dir) == len(prefix) || dir[len(prefix)] == filepath.Separator)
}

type vendoredSearch struct {
	// Path to the "vendor" directory
	vendor string

	// Path to last project identified
	lastdir string

	// Vendored packages, indexed by Root
	vendored map[string]*vcs.RepoRoot
}

func (s *vendoredSearch) inLastDir(pth string) bool {
	return s.lastdir != "" && pathStartsWith(pth, s.lastdir)
}

func processVendoredSource(src *GoSource, search *vendoredSearch, pth string) error {
	// For .go source files, see which directory they are in
	thisImport := filepath.Dir(pth[1+len(search.vendor):])
	repoRoot, err := src.RepoRootForImportPath(thisImport)
	if err != nil {
		return err
	}

	// The project name is relative to the vendor dir
	search.vendored[repoRoot.Root] = repoRoot
	search.lastdir = filepath.Join(search.vendor, repoRoot.Root)
	return nil
}

func (src GoSource) findImportPath() (string, error) {
	var importPath string
	search := func(path string, info os.FileInfo, err error) error {
		if importPath != "" {
			return filepath.SkipDir
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
	if importPath == "" {
		return "", ErrorNeedImportPath
	}
	return importPath, nil
}

// Project returns information about the project given its import
// path. If importPath is "" it is deduced from import comments, if
// available.
func (src GoSource) Project(importPath string) (*vcs.RepoRoot, error) {
	if importPath == "" {
		if src.Package == "" {
			var err error
			importPath, err = src.findImportPath()
			if err != nil {
				return nil, err
			}

			src.Package = importPath
		} else {
			importPath = src.Package
		}
	}

	repoRoot, err := vcs.RepoRootForImportPath(importPath, false)
	return repoRoot, err
}

// VendoredProjects return a map of project import names to information
// about those projects, including which version control system they use.
func (src GoSource) VendoredProjects() (map[string]*vcs.RepoRoot, error) {
	search := vendoredSearch{
		vendor:   src.Vendor(),
		vendored: make(map[string]*vcs.RepoRoot),
	}
	walkfn := func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			// Stop on error
			return err
		}

		// Ignore paths within the last project we identified
		if search.inLastDir(pth) {
			return nil
		}

		// Ignore anything except Go source
		if !info.Mode().IsRegular() || !strings.HasSuffix(pth, ".go") {
			return nil
		}

		// Identify the project
		return processVendoredSource(&src, &search, pth)
	}

	if _, err := os.Stat(src.Path); err != nil {
		return nil, err
	}

	if _, err := os.Stat(search.vendor); err == nil {
		err = filepath.Walk(search.vendor, walkfn)
		if err != nil {
			return nil, err
		}
	}

	return search.vendored, nil
}

func matchFromRefs(hashes FileHashes, wt *WorkingTree, refs []string) (string, error) {
	for _, ref := range refs {
		match, err := wt.FileHashesAreSubset(hashes, ref)
		if err != nil {
			return "", err
		}
		if match {
			return ref, nil
		}
	}

	return "", ErrorVersionNotFound
}

// Reference describes the origin of a vendored project.
type Reference struct {
	// Tag is the semver tag within the upstream repository which
	// corresponds exactly to the vendored copy of the project. If
	// no tag corresponds Tag is "".
	Tag string

	// Rev is the upstream revision from which the vendored
	// copy was taken. If this is not known Reference is "".
	Rev string

	// Ver is the semantic version or pseudo-version for the
	// commit named in Reference. This is Tag if Tag is not "".
	Ver string
}

// DescribeProject attempts to identify the tag in the version control
// system which corresponds to the project. Vendored files and files
// whose names begin with "." are ignored.
func DescribeProject(project *vcs.RepoRoot, root string) (*Reference, error) {
	wt, err := NewWorkingTree(project)
	if err != nil {
		return nil, err
	}
	defer wt.Close()

	hashes, err := NewFileHashes(wt.VCS.Cmd, root)
	if err != nil {
		return nil, err
	}

	for path, _ := range hashes {
		if strings.HasPrefix(path, "vendor/") ||
			// Ignore dot files (e.g. .git)
			strings.HasPrefix(path, ".") {
			delete(hashes, path)
		}
	}

	// First try matching against tags for semantic versions
	tags, err := wt.VersionTags()
	if err != nil {
		return nil, err
	}

	match, err := matchFromRefs(hashes, wt, tags)
	if (err != nil && err != ErrorVersionNotFound) || match != "" {
		rev, err := wt.RevisionFromTag(match)
		if err != nil {
			return nil, err
		}

		return &Reference{
			Tag: match,
			Rev: rev,
			Ver: match,
		}, nil
	}

	// Next try each revision
	revs, err := wt.Revisions()
	if err != nil {
		return nil, err
	}

	rev, err := matchFromRefs(hashes, wt, revs)
	if err != nil {
		return nil, err
	}

	ver, err := wt.PseudoVersion(rev)
	if err != nil {
		return nil, err
	}

	return &Reference{
		Rev: rev,
		Ver: ver,
	}, nil
}

// DescribeVendoredProject attempts to identify the tag in the version
// control system which corresponds to the vendored copy of the
// project.
func (src GoSource) DescribeVendoredProject(project *vcs.RepoRoot) (*Reference, error) {
	projectdir := filepath.Join(src.Vendor(), project.Root)
	ref, err := DescribeProject(project, projectdir)
	return ref, err
}
