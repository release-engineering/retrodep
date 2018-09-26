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
		filePath, importPath string
		ok                   bool
	}{
		{
			"/home/foo/github.com/release-engineering/backvendor",
			"github.com/release-engineering/backvendor",
			true,
		},
		{
			"/home/foo/github.com/release-engineering/backvendor/",
			"github.com/release-engineering/backvendor",
			true,
		},
		{
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
			t.Errorf("wrong ok value for %s: got _,%v, want _,%v",
				test.filePath, ok, test.ok)
			continue
		}
		if !ok {
			continue
		}

		if importPath != test.importPath {
			t.Errorf("wrong path for %s: got %q, want %q",
				test.filePath, importPath, test.importPath)
		}
	}
}
