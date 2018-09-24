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

func TestDirs(t *testing.T) {
	src, err := NewGoSource("testdata/gosource")
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
	src, err := NewGoSource("testdata/gosource")
	if err != nil {
		t.Fatal(err)
	}
	if src.usesGodep {
		t.Fatal("usesGodep")
	}
}

func TestGodepTrue(t *testing.T) {
	src, err := NewGoSource("testdata/godep")
	if err != nil {
		t.Fatal(err)
	}
	if !src.usesGodep {
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
