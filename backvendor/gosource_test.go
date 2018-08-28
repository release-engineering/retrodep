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
	"fmt"
	"testing"
)

func TestDirs(t *testing.T) {
	src := GoSource("testdata/gosource")
	if src.Topdir() != "testdata/gosource" {
		t.Fatal("Topdir")
	}
	if src.Vendor() != "testdata/gosource/vendor" {
		t.Fatal("Vendor")
	}
}

func TestVendoredProjects(t *testing.T) {
	src := GoSource("testdata/gosource")
	expected := []string{
		"github.com/eggs/ham",
		"github.com/foo/bar",
	}
	got, err := src.VendoredProjects()
	if err != nil {
		t.Fatal(err)
	}
	for each := range got {
		fmt.Printf("%v\n", each)
	}
	matched := len(got) == len(expected)
	if !matched {
		t.Errorf("%d != %d", len(got), len(expected))
	}
	if matched {
		for _, repo := range expected {
			if _, ok := got[repo]; !ok {
				t.Errorf("%s not returned", repo)
				matched = false
				break
			}
		}
	}
	if !matched {
		t.Errorf("%v != %v", got, expected)
	}
}
