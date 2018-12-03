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

package retrodep

import (
	"testing"
)

func TestVendoredProjects(t *testing.T) {
	src, err := NewGoSource("testdata/gosource", nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"github.com/eggs/ham",
		"github.com/foo/bar",
	}
	got, err := src.VendoredProjects()
	if err != nil {
		t.Fatal(err)
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

func TestChooseBestTag(t *testing.T) {
	tags := []string{
		"1.2.3-beta1",
		"1.2.2",
		"1.2.2-beta2",
	}
	best := chooseBestTag(tags)
	if best != "1.2.2" {
		t.Errorf("wrong best tag (%s)", best)
	}
}
