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
	"strings"
	"testing"

	"golang.org/x/tools/go/vcs"
)

func TestGitRevisions(t *testing.T) {
	defer mockExecCommand()()

	wt := gitWorkingTree{
		anyWorkingTree: anyWorkingTree{
			Source: &GoSource{},
			VCS:    vcs.ByCmd("git"),
		},
	}

	expectedRevs := []string{
		"d4c3dbfa77a74ae238e401d5d2197b45f30d8513",
		"a2176f4275f92ceddb47cff1e363313156124bf6",
	}
	mockedStdout = strings.Join(expectedRevs, "\n") + "\n"

	revs, err := wt.Revisions()
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != len(expectedRevs) {
		t.Fatalf("wrong number of revisions: got %d, want %d",
			len(revs), len(expectedRevs))
	}
	for i, rev := range expectedRevs {
		if revs[i] != rev {
			t.Fatalf("unexpected revisions: got %v, want %v",
				revs, expectedRevs)
		}
	}
}

func TestGitRevisionsError(t *testing.T) {
	defer mockExecCommand()()

	wt := gitWorkingTree{
		anyWorkingTree: anyWorkingTree{
			Source: &GoSource{},
			VCS:    vcs.ByCmd("git"),
		},
	}

	mockedStderr = "fatal: not a git repository\n"
	mockedExitStatus = 128

	_, err := wt.Revisions()
	if err == nil {
		t.Fatal("git failure was not reported")
	}
}
