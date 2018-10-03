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
	"os"
	"testing"

	"golang.org/x/tools/go/vcs"
)

func TestFindExcludes(t *testing.T) {
	type tcase struct {
		dir   string
		globs []string
		exp   []string
	}
	tcases := []tcase{
		tcase{
			dir:   "testdata/gosource",
			globs: nil,
			exp:   []string{},
		},

		tcase{
			dir:   "testdata/gosource",
			globs: []string{"vendor*"},
			exp:   []string{"testdata/gosource/vendor"},
		},
	}
	for _, tc := range tcases {
		excl, err := FindExcludes(tc.dir, tc.globs)
		if err != nil {
			t.Fatal(err)
		}
		if len(tc.exp) != len(excl) {
			t.Errorf("wrong length: got %d, want %d", len(excl), len(tc.exp))
		}
		for i, e := range tc.exp {
			if excl[i] != e {
				t.Errorf("wrong value: got %v, want %v", excl, tc.exp)
				break
			}
		}
	}
}

func TestNewGoSource(t *testing.T) {
	type tcase struct {
		path  string
		pkg   string
		expOk bool
	}
	tcases := []tcase{
		tcase{"testdata/gosource", "", true},
		tcase{"testdata/godep", "example.com/godep", true},
		tcase{"testdata", "", false},
	}
	for _, tc := range tcases {
		s, err := NewGoSource(tc.path, nil)
		ok := err == nil
		if ok != tc.expOk {
			t.Errorf("%s: got %s, want ok:%t",
				tc.path, err, tc.expOk)
		}

		if err != nil {
			continue
		}

		if s.Package != tc.pkg {
			t.Errorf("%s: got package %q, want %q",
				tc.path, s.Package, tc.pkg)
		}
	}
}

func TestFindGoSources(t *testing.T) {
	type exp struct {
		path, subpath string
	}
	type tcase struct {
		name string
		path string
		exp  []exp
	}
	tcases := []tcase{
		tcase{
			name: "single",
			path: "testdata/gosource",
			exp:  []exp{{"testdata/gosource", ""}},
		},

		tcase{
			name: "multi",
			path: "testdata/multi",
			exp: []exp{
				{"testdata/multi/abc", "abc"},
				{"testdata/multi/def", "def"},
			},
		},
	}
	for _, tc := range tcases {
		srcs, err := FindGoSources(tc.path, nil)
		if err != nil {
			t.Errorf("%s: %s", tc.name, err)
			continue
		}
		if srcs == nil {
			t.Errorf("%s: srcs is nil", tc.name)
			continue
		}
		if len(srcs) != len(tc.exp) {
			t.Errorf("%s: got %d sources, want %d", tc.name, len(srcs), len(tc.exp))
			continue
		}
		for i, src := range tc.exp {
			if src.path != srcs[i].Path {
				t.Errorf("%s: Path: got %q, want %q", tc.name, srcs[i].Path, src.path)
			}
			if src.subpath != srcs[i].SubPath {
				t.Errorf("%s: SubPath: got %q, want %q", tc.name, srcs[i].SubPath, src.subpath)
			}
		}
	}
}

func TestProject(t *testing.T) {
	type tcase struct {
		name       string
		importPath string
		root       string
		expSubPath string
	}
	tcases := []tcase{
		tcase{
			name:       "trivial",
			importPath: "example.com/foo",
			root:       "example.com/foo",
			expSubPath: "",
		},

		tcase{
			name:       "subdir",
			importPath: "example.com/foo/bar",
			root:       "example.com/foo",
			expSubPath: "bar",
		},
	}
	src, err := NewGoSource("testdata/gosource", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Reset vcsRepoRootForImportPath after this test.
	defer func() {
		vcsRepoRootForImportPath = vcs.RepoRootForImportPath
	}()

	for _, tc := range tcases {
		vcsRepoRootForImportPath = func(importPath string, _ bool) (*vcs.RepoRoot, error) {
			return &vcs.RepoRoot{
				Root: tc.root,
			}, nil
		}

		repoPath, err := src.Project(tc.importPath)
		if err != nil {
			t.Errorf("%s: %s", tc.name, err)
			continue
		}
		if repoPath.SubPath != tc.expSubPath {
			t.Errorf("%s: SubPath: want %q, got %q", tc.name, repoPath.SubPath,
				tc.expSubPath)
		}
	}
}

func TestDirs(t *testing.T) {
	src, err := NewGoSource("testdata/gosource", nil)
	if err != nil {
		t.Fatal(err)
	}
	if src.Path != "testdata/gosource" {
		t.Fatal("Path")
	}
	if src.Vendor() != "testdata/gosource/vendor" {
		t.Fatal("Vendor")
	}
}

func TestGodepFalse(t *testing.T) {
	src, err := NewGoSource("testdata/gosource", nil)
	if err != nil {
		t.Fatal(err)
	}
	if src.usesGodep {
		t.Fatal("usesGodep")
	}
}

func TestGodepTrue(t *testing.T) {
	src, err := NewGoSource("testdata/godep", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !src.usesGodep {
		t.Fatal("usesGodep")
	}
	exp := "example.com/godep"
	if src.Package != exp {
		t.Errorf("wrong import path detected: want %s, got %s",
			exp, src.Package)
	}
}

func TestGlideFalse(t *testing.T) {
	src, err := NewGoSource("testdata/godep", nil)
	if err != nil {
		t.Fatal(err)
	}
	if src.Package == "github.com/release-engineering/backvendor/testdata/glide" {
		t.Fatal("usesGlide")
	}
}

func TestGlideTrue(t *testing.T) {
	src, err := NewGoSource("testdata/glide", nil)
	if err != nil {
		t.Fatal(err)
	}
	if src.Package != "github.com/release-engineering/backvendor/testdata/glide" {
		t.Fatal("usesGodep")
	}
}

func TestImportPathFromFilepath(t *testing.T) {
	tests := []struct {
		name                 string
		filePath, importPath string
		ok                   bool
	}{
		{
			"toplevel",
			"/home/foo/github.com/release-engineering/backvendor",
			"github.com/release-engineering/backvendor",
			true,
		},
		{
			"subdir",
			"/home/foo/github.com/release-engineering/backvendor/backvendor",
			"github.com/release-engineering/backvendor/backvendor",
			true,
		},
		{
			"trailing-slash",
			"/home/foo/github.com/release-engineering/backvendor/",
			"github.com/release-engineering/backvendor",
			true,
		},
		{
			"unknown",
			"release-engineering/backvendor",
			"",
			false,
		},
	}

	// Start in the root directory to make sure Abs doesn't figure
	// anything out from the path to the project we're in.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	err = os.Chdir("/")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		importPath, ok := importPathFromFilepath(test.filePath)
		if ok != test.ok {
			t.Errorf("%s: wrong ok value for %s: got _,%v, want _,%v",
				test.name, test.filePath, ok, test.ok)
			continue
		}
		if !ok {
			continue
		}

		if importPath != test.importPath {
			t.Errorf("%s: wrong path for %s: got %q, want %q",
				test.name, test.filePath, importPath, test.importPath)
		}
	}
}
