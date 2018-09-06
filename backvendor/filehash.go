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
type FileHashes map[string]FileHash

func hash(vcscmd, relativepath, abspath string) (FileHash, error) {
	if vcscmd != vcsGit.Cmd {
		return FileHash(""), ErrorUnknownVCS
	}

	args := []string{"hash-object", "--path", relativepath, abspath}
	cmd := exec.Command(vcscmd, args...)
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
// whose files belong to the version control system named in vcscmd.
func NewFileHashes(vcscmd, root string) (FileHashes, error) {
	hashes := make(FileHashes)
	root = path.Clean(root)
	var rootlen int
	switch {
	case root == ".":
		rootlen = 0
	default:
		rootlen = 1 + len(root)
	}
	ignore := make(map[string]struct{}) // set of pathnames to ignore
	walkfn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, ok := ignore[path]; ok {
			// This pathname has been ignored due to .gitattributes
			return nil
		}
		if info.IsDir() {
			// Check for .gitattributes in this directory
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
						ignore[fn] = struct{}{}
						break
					}
				}
			}

			return nil
		}
		relativepath := path[rootlen:]
		filehash, err := hash(vcscmd, relativepath, path)
		if err != nil {
			return err
		}
		hashes[relativepath] = filehash
		return nil
	}
	err := filepath.Walk(root, walkfn)
	if err != nil {
		return nil, err
	}
	return hashes, nil
}
