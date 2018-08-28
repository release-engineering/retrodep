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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"golang.org/x/tools/go/vcs"
)

// A WorkingTree is a local checkout of Go source code, and
// information about the version control system it came from.
type WorkingTree struct {
	Source GoSource
	VCS    *vcs.Cmd
}

// NewWorkingTree creates a local checkout of the version control
// system for a Go project.
func NewWorkingTree(project *vcs.RepoRoot) (*WorkingTree, error) {
	dir, err := ioutil.TempDir("", "go-backvendor.")
	if err != nil {
		return nil, err
	}
	err = project.VCS.Create(dir, project.Repo)
	if err != nil {
		return nil, err
	}

	return &WorkingTree{
		Source: GoSource(dir),
		VCS:    project.VCS,
	}, nil
}

// Close removes the local checkout.
func (wt *WorkingTree) Close() error {
	return os.RemoveAll(wt.Source.Topdir())
}

// SemVerTags returns a list of the semantic tags, i.e. those tags which are
// parseable as semantic tags such as v1.1.0.
func (wt *WorkingTree) SemVerTags() ([]string, error) {
	tags, err := wt.VCS.Tags(wt.Source.Topdir())
	if err != nil {
		return nil, err
	}
	semvers := make(semver.Collection, 0)
	semvertags := make(map[*semver.Version]string)
	for _, tag := range tags {
		v, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		semvers = append(semvers, v)
		semvertags[v] = tag
	}
	sort.Sort(sort.Reverse(semvers))
	strtags := make([]string, len(semvers))
	for i, v := range semvers {
		strtags[i] = semvertags[v]
	}
	return strtags, nil
}

// Revisions returns all revisions in the repository.
func (wt *WorkingTree) Revisions() ([]string, error) {
	if wt.VCS.Cmd != vcsGit {
		return nil, ErrorUnknownVCS
	}

	args := []string{"rev-list", "--all"}
	cmd := exec.Command(wt.VCS.Cmd, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Dir = wt.Source.Topdir()
	err := cmd.Run()
	if err != nil {
		os.Stderr.Write(buf.Bytes())
		return nil, err
	}
	revisions := make([]string, 0)
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		revisions = append(revisions, strings.TrimSpace(scanner.Text()))
	}
	return revisions, nil
}

func (wt *WorkingTree) RevisionFromTag(tag string) (rev string, err error) {
	if wt.VCS.Cmd != vcsGit {
		err = ErrorUnknownVCS
		return
	}

	args := []string{"rev-parse", tag}
	cmd := exec.Command(wt.VCS.Cmd, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Dir = wt.Source.Topdir()
	err = cmd.Run()
	if err != nil {
		os.Stderr.Write(buf.Bytes())
		return
	}
	rev = strings.TrimSpace(buf.String())
	return
}

// DescribeRevision returns a name to describe a particular revision,
// or the error ErrorVersionNotFound if no such name is available.
func (wt *WorkingTree) DescribeRevision(rev string) (desc string, err error) {
	if wt.VCS.Cmd != vcsGit {
		err = ErrorUnknownVCS
		return
	}

	args := []string{"describe", "--tags", rev}
	cmd := exec.Command(wt.VCS.Cmd, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Dir = wt.Source.Topdir()
	err = cmd.Run()
	if err != nil {
		output := strings.TrimSpace(buf.String())
		if output == "fatal: No names found, cannot describe anything." ||
			strings.HasPrefix(output, "fatal: No tags can describe ") {
			err = ErrorVersionNotFound
			return
		}

		os.Stderr.Write(buf.Bytes())
		return
	}
	desc = strings.TrimSpace(buf.String())
	return
}

// FileHashesAreSubset compares a set of files and their hashes with
// those from a particular tag. It returns true if the provided files
// and hashes are a subset of those found at the tag.
func (wt *WorkingTree) FileHashesAreSubset(fh FileHashes, tag string) (bool, error) {
	if wt.VCS.Cmd != vcsGit {
		return false, ErrorUnknownVCS
	}

	args := []string{"ls-tree", "-r", tag}
	cmd := exec.Command(wt.VCS.Cmd, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Dir = wt.Source.Topdir()
	err := cmd.Run()
	if err != nil {
		if strings.HasPrefix(buf.String(), "fatal: Not a valid object name ") {
			// This is a branch name, not a tag name
			return false, nil
		}

		os.Stderr.Write(buf.Bytes())
		return false, err
	}
	tagfilehashes := make(FileHashes)
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			return false, fmt.Errorf("line not understood: %s", line)
		}

		var mode uint32
		if _, err = fmt.Sscanf(fields[0], "%o", &mode); err != nil {
			return false, err
		}
		tagfilehashes[fields[3]] = FileHash(fields[2])
	}
	for path, filehash := range fh {
		tagfilehash, ok := tagfilehashes[path]
		if !ok {
			// File not present in tag
			return false, nil
		}
		if filehash != tagfilehash {
			// Hash does not match
			return false, nil
		}
	}
	return true, nil
}
