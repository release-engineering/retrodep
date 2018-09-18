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

// This file contains methods specific to working with git.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// gitRevisions returns all revisions in the git repository, using
// 'git rev-list --all'.
func (wt *WorkingTree) gitRevisions() ([]string, error) {
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

// gitRevisionFromTag returns the commit hash for the given tag, using
// 'git rev-parse ...'
func (wt *WorkingTree) gitRevisionFromTag(tag string) (string, error) {
	buf, err := wt.run("rev-parse", tag)
	if err != nil {
		os.Stderr.Write(buf.Bytes())
		return "", err
	}
	rev := strings.TrimSpace(buf.String())
	return rev, nil
}

// gitRevSync updates the working tree to reflect the tag or revision
// ref, using 'git checkout ...'. The working tree must not have been
// locally modified.
func (wt *WorkingTree) gitRevSync(ref string) error {
	buf, err := wt.run("checkout", ref)
	if err != nil {
		os.Stderr.Write(buf.Bytes())
	}
	return err
}

// gitTimeFromRevision returns the commit timestamp for the revision
// rev, using 'git show -s --pretty=format:%cI ...'.
func (wt *WorkingTree) gitTimeFromRev(rev string) (time.Time, error) {
	var t time.Time
	buf, err := wt.run("show", "-s", "--pretty=format:%cI", rev)
	if err != nil {
		return t, err
	}

	t, err = time.Parse(time.RFC3339, strings.TrimSpace(buf.String()))
	return t, err
}

// gitReachableTag returns the most recent reachable semver tag, using
// 'git describe --tags --match=...', with match globs for tags that
// are likely to be semvers. It returns ErrorVersionNotFound if no
// suitable tag is found.
func (wt *WorkingTree) gitReachableTag(rev string) (string, error) {
	var tag string
	for _, match := range []string{"v[0-9]*", "[0-9]*"} {
		buf, err := wt.run("describe", "--tags", "--match="+match, rev)
		output := strings.TrimSpace(buf.String())
		if err == nil {
			tag = output
			break
		}

		// Catch failures due to not finding an appropriate tag
		output = strings.ToLower(output)
		switch {
		// fatal: no tag exactly matches ...
		// fatal: no tags can describe ...
		// fatal: no names found, cannot describe anything.
		// fatal: no annotated tags can describe ...
		case strings.HasPrefix(output, "fatal: no tag"),
			strings.HasPrefix(output, "fatal: no names"),
			strings.HasPrefix(output, "fatal: no annotated tag"):
			err = ErrorVersionNotFound
		default:
			os.Stderr.Write(buf.Bytes())
		}
		return "", err
	}

	if tag == "" {
		return "", ErrorVersionNotFound
	}

	log.Debugf("%s is described as %s", rev, tag)
	fields := strings.Split(tag, "-")
	if len(fields) < 3 {
		// This matches a tag exactly (it must not be a semver tag)
		return tag, nil
	}
	tag = strings.Join(fields[:len(fields)-2], "-")
	return tag, nil
}

// gitFileHashesFromRef parses the output of 'git ls-tree -r' to
// return the file hashes for the given tag or revision ref.
func (wt *WorkingTree) gitFileHashesFromRef(ref string) (*FileHashes, error) {
	buf, err := wt.run("ls-tree", "-r", ref)
	if err != nil {
		if strings.HasPrefix(buf.String(), "fatal: Not a valid object name ") {
			// This is a branch name, not a tag name
			return nil, ErrorInvalidRef
		}

		os.Stderr.Write(buf.Bytes())
		return nil, err
	}
	fh := make(map[string]FileHash)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			return nil, fmt.Errorf("line not understood: %s", line)
		}

		var mode uint32
		if _, err = fmt.Sscanf(fields[0], "%o", &mode); err != nil {
			return nil, err
		}
		fh[fields[3]] = FileHash(fields[2])
	}

	return &FileHashes{
		vcsCmd: vcsGit,
		root:   wt.Source.Path,
		hashes: fh,
	}, nil
}
