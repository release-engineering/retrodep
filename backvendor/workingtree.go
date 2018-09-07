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
	"time"

	"github.com/Masterminds/semver"
	"golang.org/x/tools/go/vcs"
)

// A WorkingTree is a local checkout of Go source code, and
// information about the version control system it came from.
type WorkingTree struct {
	Source *GoSource
	VCS    *vcs.Cmd
}

// NewWorkingTree creates a local checkout of the version control
// system for a Go project.
func NewWorkingTree(project *vcs.RepoRoot) (*WorkingTree, error) {
	dir, err := ioutil.TempDir("", "backvendor.")
	if err != nil {
		return nil, err
	}
	err = project.VCS.Create(dir, project.Repo)
	if err != nil {
		return nil, err
	}

	return &WorkingTree{
		Source: NewGoSource(dir),
		VCS:    project.VCS,
	}, nil
}

// Close removes the local checkout.
func (wt *WorkingTree) Close() error {
	return os.RemoveAll(wt.Source.Path)
}

// VersionTags returns the tags that are parseable as semantic tags,
// e.g. v1.1.0.
func (wt *WorkingTree) VersionTags() ([]string, error) {
	tags, err := wt.VCS.Tags(wt.Source.Path)
	if err != nil {
		return nil, err
	}
	versions := make(semver.Collection, 0)
	versionTags := make(map[*semver.Version]string)
	for _, tag := range tags {
		v, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		versions = append(versions, v)
		versionTags[v] = tag
	}
	sort.Sort(sort.Reverse(versions))
	strTags := make([]string, len(versions))
	for i, v := range versions {
		strTags[i] = versionTags[v]
	}
	return strTags, nil
}

// run runs the VCS command with the provided args
// and returns a bytes.Buffer.
func (wt *WorkingTree) run(args ...string) (*bytes.Buffer, error) {
	p := exec.Command(wt.VCS.Cmd, args...)
	var buf bytes.Buffer
	p.Stdout = &buf
	p.Stderr = &buf
	p.Dir = wt.Source.Path
	err := p.Run()
	return &buf, err
}

// Revisions returns all revisions in the repository.
func (wt *WorkingTree) Revisions() ([]string, error) {
	if wt.VCS.Cmd != vcsGit.Cmd {
		return nil, ErrorUnknownVCS
	}

	buf, err := wt.run("rev-list", "--all")
	if err != nil {
		os.Stderr.Write(buf.Bytes())
		return nil, err
	}
	revisions := make([]string, 0)
	output := bufio.NewScanner(buf)
	for output.Scan() {
		revisions = append(revisions, strings.TrimSpace(output.Text()))
	}
	return revisions, nil
}

func (wt *WorkingTree) RevisionFromTag(tag string) (string, error) {
	if wt.VCS.Cmd != vcsGit.Cmd {
		return "", ErrorUnknownVCS
	}

	buf, err := wt.run("rev-parse", tag)
	if err != nil {
		os.Stderr.Write(buf.Bytes())
		return "", err
	}
	rev := strings.TrimSpace(buf.String())
	return rev, nil
}

func (wt *WorkingTree) timeFromRevision(rev string) (time.Time, error) {
	var t time.Time
	if wt.VCS.Cmd != vcsGit.Cmd {
		return t, ErrorUnknownVCS
	}

	buf, err := wt.run("show", "-s", "--pretty=format:%cI", rev)
	if err != nil {
		return t, err
	}

	t, err = time.Parse(time.RFC3339, strings.TrimSpace(buf.String()))
	return t, err
}

// reachableTag returns the most recent reachable semver tag.
func (wt *WorkingTree) reachableTag(rev string) (string, error) {
	if wt.VCS.Cmd != vcsGit.Cmd {
		return "", ErrorUnknownVCS
	}

	var tag string
	for _, match := range []string{"v[0-9]*", "[0-9]*"} {
		buf, err := wt.run("describe", "--match="+match, rev)
		output := strings.TrimSpace(buf.String())
		if err == nil {
			tag = output
			break
		}

		if output != "fatal: No names found, cannot describe anything." &&
			!strings.HasPrefix(output, "fatal: No annotated tags can describe ") &&
			!strings.HasPrefix(output, "fatal: No tags can describe ") {
			os.Stderr.Write(buf.Bytes())
			return "", err
		}
	}

	if tag == "" {
		return "", ErrorVersionNotFound
	}

	fields := strings.Split(tag, "-")
	if len(fields) < 3 {
		return "", fmt.Errorf("too few dashes: %s", tag)
	}
	tag = strings.Join(fields[:len(fields)-2], "")
	return tag, nil
}

func (wt *WorkingTree) PseudoVersion(rev string) (string, error) {
	if wt.VCS.Cmd != vcsGit.Cmd {
		return "", ErrorUnknownVCS
	}

	suffix := "-0." // This commit is *before* some other tag
	var version string
	reachable, err := wt.reachableTag(rev)
	if err == ErrorVersionNotFound {
		version = "v0.0.0"
	} else if err != nil {
		return "", err
	} else {
		ver, err := semver.NewVersion(reachable)
		if err != nil {
			// Not a semantic version. Use a timestamped suffix
			// to indicate this commit is *after* the tag
			version = reachable
			suffix = "-1."
		} else {
			if ver.Prerelease() == "" {
				*ver = ver.IncPatch()
			} else {
				suffix = ".0."
			}

			version = "v" + ver.String()
		}
	}

	t, err := wt.timeFromRevision(rev)
	if err != nil {
		return "", err
	}

	timestamp := t.Format("20060102150405")
	pseudo := version + suffix + timestamp + "-" + rev[:12]
	return pseudo, nil
}

// FileHashesAreSubset compares a set of files and their hashes with
// those from a particular tag. It returns true if the provided files
// and hashes are a subset of those found at the tag.
func (wt *WorkingTree) FileHashesAreSubset(fh FileHashes, tag string) (bool, error) {
	if wt.VCS.Cmd != vcsGit.Cmd {
		return false, ErrorUnknownVCS
	}

	buf, err := wt.run("ls-tree", "-r", tag)
	if err != nil {
		if strings.HasPrefix(buf.String(), "fatal: Not a valid object name ") {
			// This is a branch name, not a tag name
			return false, nil
		}

		os.Stderr.Write(buf.Bytes())
		return false, err
	}
	tagFileHashes := make(FileHashes)
	scanner := bufio.NewScanner(buf)
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
		tagFileHashes[fields[3]] = FileHash(fields[2])
	}
	for path, fileHash := range fh {
		tagFileHash, ok := tagFileHashes[path]
		if !ok {
			// File not present in tag
			return false, nil
		}
		if fileHash != tagFileHash {
			// Hash does not match
			return false, nil
		}
	}
	return true, nil
}
