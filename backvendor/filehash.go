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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// FileHash records the hash of a file in the format preferred by the
// version control system that tracks it.
type FileHash string

// FileHashes is a map of paths, relative to the top-level of the
// version control system, to their hashes.
type FileHashes struct {
	vcsCmd, root string
	hashes       map[string]FileHash
}

func hashFile(vcsCmd, relativePath, absPath string) (FileHash, error) {
	if vcsCmd != vcsGit {
		return FileHash(""), ErrorUnknownVCS
	}

	args := []string{"hash-object", "--path", relativePath, absPath}
	cmd := exec.Command(vcsCmd, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		os.Stderr.Write(buf.Bytes())
		return FileHash(""), err
	}

	return FileHash(strings.TrimSpace(buf.String())), nil
}

// NewFileHashes returns a new FileHashes from a filesystem tree at root,
// whose files belong to the version control system named in vcsCmd. Keys in
// the excludes map are filenames to ignore.
func NewFileHashes(vcsCmd, root string, excludes map[string]struct{}) (*FileHashes, error) {
	hashes := &FileHashes{
		vcsCmd: vcsCmd,
		root:   root,
		hashes: make(map[string]FileHash),
	}
	root = path.Clean(root)
	var rootlen int
	switch {
	case root == ".":
		rootlen = 0
	default:
		rootlen = 1 + len(root)
	}

	// Make a local copy of excludes we can safely modify
	excl := make(map[string]struct{})
	if excludes != nil {
		for k, v := range excludes {
			excl[k] = v
		}
	}

	walkfn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, skip := excl[path]; skip {
			// This pathname has been ignored, either by caller
			// request or due to .gitattributes
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			// Check for .gitattributes in this directory
			// FIXME: gitattributes(5) describes a more complex file
			// format than handled here.  Can git-check-attr(1) help?
			ga, err := os.Open(filepath.Join(path, ".gitattributes"))
			if err != nil {
				return nil
			}
			defer ga.Close()

			scanner := bufio.NewScanner(bufio.NewReader(ga))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) < 2 {
					continue
				}
				for _, field := range fields[1:] {
					if field == "export-subst" {
						// Not expected to have matching hash
						fn := filepath.Join(path, fields[0])
						excl[fn] = struct{}{}
						break
					}
				}
			}

			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relativePath := path[rootlen:]
		fileHash, err := hashFile(vcsCmd, relativePath, path)
		if err != nil {
			return err
		}
		hashes.hashes[relativePath] = fileHash
		return nil
	}
	err := filepath.Walk(root, walkfn)
	if err != nil {
		return nil, err
	}
	return hashes, nil
}

// IsSubsetOf returns true if these file hashes are a subset of s.
func (h *FileHashes) IsSubsetOf(s *FileHashes) bool {
	for path, fileHash := range h.hashes {
		sh, ok := s.hashes[path]
		if !ok {
			// File not present in tag
			log.Debugf("%s: not present", path)
			return false
		}
		if fileHash != sh {
			// Hash does not match
			log.Debugf("%s: hash mismatch", path)
			return false
		}
	}

	return true
}
