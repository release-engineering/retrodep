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
	"path/filepath"

	"github.com/pkg/errors"

	"golang.org/x/tools/go/vcs"
)

// VendoredProjects return a map of project import names to information
// about those projects, including which version control system they use.
func (src *GoSource) VendoredProjects() (map[string]*vcs.RepoRoot, error) {
	mainpkgs, err := src.Load()
	if err != nil {
		return nil, err
	}
	var toppkg string
	for _, mainpkg := range mainpkgs {
		if toppkg == "" || len(mainpkg.PkgPath) < len(toppkg) {
			toppkg = mainpkg.PkgPath
		}
	}
	vendored := make(map[string]*vcs.RepoRoot)
	for _, mainpkg := range mainpkgs {
		for importPath, imp := range mainpkg.Imports {
			if imp.ID == importPath {
				continue
			}

			reporoot, err := vcs.RepoRootForImportPath(importPath, false)
			if err != nil {
				return nil, errors.Wrap(err,
					"from RepoRootForImportPath")
			}
			vendored[reporoot.Root] = reporoot
		}
	}
	return vendored, nil
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

// DescribeVendoredProject attempts to identify the tag in the version
// control system which corresponds to the vendored copy of the
// project.
func (src *GoSource) DescribeVendoredProject(project *vcs.RepoRoot) (*Reference, error) {
	wt, err := NewWorkingTree(project)
	if err != nil {
		return nil, err
	}
	defer wt.Close()

	projectdir := filepath.Join(src.Vendor(), project.Root)
	hashes, err := NewFileHashes(wt.VCS.Cmd, projectdir)
	if err != nil {
		return nil, err
	}

	// First try matching against tags for semantic versions
	tags, err := wt.SemVerTags()
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
